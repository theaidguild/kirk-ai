/**
 * termynal.js
 * A lightweight, modern and flexible animated terminal window, using
 * async/await.
 *
 * @author Ines Montani <ines@ines.io>
 * @version 0.0.1
 * @license MIT
 */

'use strict';

/** Generate a terminal widget. */
class Termynal {
    /**
     * Construct the widget's settings.
     * @param {(string|Node)=} container - Query selector or container element.
     * @param {Object=} options - Custom settings.
     * @param {string} options.prefix - Prefix to use for data attributes.
     * @param {number} options.startDelay - Delay before animation, in ms.
     * @param {number} options.typeDelay - Delay between each typed character, in ms.
     * @param {number} options.lineDelay - Delay between each line, in ms.
     * @param {number} options.progressLength - Number of characters displayed as progress bar.
     * @param {string} options.progressChar – Character to use for progress bar, defaults to █.
     * @param {number} options.progressPercent - Max percent of progress.
     * @param {string} options.cursor – Character to use for cursor, defaults to ▋.
     * @param {Object[]} lineData - Dynamically loaded line data objects.
     * @param {boolean} options.noInit - Don't initialise the animation.
     */
    constructor(container = '#termynal', options = {}) {
        this.container = (typeof container === 'string') ? document.querySelector(container) : container;
        this.pfx = `data-${options.prefix || 'ty'}`;
        this.originalTypeDelay = options.typeDelay
            || parseFloat(this.container.getAttribute(`${this.pfx}-typeDelay`)) || 90;
        this.originalLineDelay = options.lineDelay
            || parseFloat(this.container.getAttribute(`${this.pfx}-lineDelay`)) || 1500;
        this.typeDelay = this.originalTypeDelay;
        this.lineDelay = this.originalLineDelay;
        this.startDelay = options.startDelay
            || parseFloat(this.container.getAttribute(`${this.pfx}-startDelay`)) || 600;
        this.progressLength = options.progressLength
            || parseFloat(this.container.getAttribute(`${this.pfx}-progressLength`)) || 40;
        this.progressChar = options.progressChar
            || this.container.getAttribute(`${this.pfx}-progressChar`) || '█';
        this.progressPercent = options.progressPercent
            || parseFloat(this.container.getAttribute(`${this.pfx}-progressPercent`)) || 100;
        this.cursor = options.cursor
            || this.container.getAttribute(`${this.pfx}-cursor`) || '▋';
        this.lineData = this.lineDataToElements(options.lineData || []);
        
        // Animation control state
        this.isRunning = false;
        this.isPaused = false;
        this.currentLineIndex = 0;
        this.speedMultiplier = 1;
        
        if (!options.noInit) this.init()
    }

    /**
     * Initialise the widget, get lines, clear container and start animation.
     */
    init() {
        // Appends dynamically loaded lines to existing line elements.
        this.lines = [...this.container.querySelectorAll(`[${this.pfx}]`)].concat(this.lineData);

        /**
         * Calculates width and height of Termynal container.
         * If container is empty and lines are dynamically loaded, defaults to browser `auto` or CSS.
         */
        const containerStyle = getComputedStyle(this.container);
        this.container.style.width = containerStyle.width !== '0px' ?
            containerStyle.width : undefined;
        this.container.style.minHeight = containerStyle.height !== '0px' ?
            containerStyle.height : undefined;

        this.container.setAttribute('data-termynal', '');
        this.container.innerHTML = '';
        this.start();
    }

    /**
     * Start the animation and render the lines depending on their data attributes.
     */
    async start() {
        if (this.isRunning) return;
        this.isRunning = true;
        this.isPaused = false;
        this.currentLineIndex = 0;
        
        await this._wait(this.startDelay);

        for (let i = 0; i < this.lines.length; i++) {
            if (!this.isRunning) break;
            
            // Wait if paused
            while (this.isPaused && this.isRunning) {
                await this._wait(100);
            }
            
            const line = this.lines[i];
            this.currentLineIndex = i;
            const type = line.getAttribute(this.pfx);
            const delay = (line.getAttribute(`${this.pfx}-delay`) || this.lineDelay) / this.speedMultiplier;

            if (type == 'input') {
                line.setAttribute(`${this.pfx}-cursor`, this.cursor);
                await this.type(line);
                await this._wait(delay);
            }

            else if (type == 'progress') {
                await this.progress(line);
                await this._wait(delay);
            }

            else {
                this.container.appendChild(line);
                await this._wait(delay);
            }

            line.removeAttribute(`${this.pfx}-cursor`);
        }
        
        this.isRunning = false;
        this.currentLineIndex = 0;
    }

    /**
     * Animate a typed line.
     * @param {Node} line - The line element to render.
     */
    async type(line) {
        const chars = [...line.textContent];
        const delay = (line.getAttribute(`${this.pfx}-typeDelay`) || this.typeDelay) / this.speedMultiplier;
        line.textContent = '';
        this.container.appendChild(line);

        for (let char of chars) {
            if (!this.isRunning) break;
            while (this.isPaused && this.isRunning) {
                await this._wait(100);
            }
            await this._wait(delay);
            line.textContent += char;
        }
    }

    /**
     * Animate a progress bar.
     * @param {Node} line - The line element to render.
     */
    async progress(line) {
        const progressLength = line.getAttribute(`${this.pfx}-progressLength`)
            || this.progressLength;
        const progressChar = line.getAttribute(`${this.pfx}-progressChar`)
            || this.progressChar;
        const chars = progressChar.repeat(progressLength);
        const progressPercent = line.getAttribute(`${this.pfx}-progressPercent`)
            || this.progressPercent;
        line.textContent = '';
        this.container.appendChild(line);

        for (let i = 1; i < chars.length + 1; i++) {
            if (!this.isRunning) break;
            while (this.isPaused && this.isRunning) {
                await this._wait(100);
            }
            await this._wait(this.typeDelay / this.speedMultiplier);
            const percent = Math.round(i / chars.length * 100);
            line.textContent = `${chars.slice(0, i)} ${percent}%`;
            if (percent>progressPercent) {
                break;
            }
        }
    }

    /**
     * Reset the terminal to initial state.
     */
    reset() {
        this.isRunning = false;
        this.isPaused = false;
        this.currentLineIndex = 0;
        this.container.innerHTML = '';
        this.speedMultiplier = 1;
    }

    /**
     * Replay the animation from the beginning.
     */
    async replay() {
        this.reset();
        await this._wait(100); // Brief pause before restart
        this.start();
    }

    /**
     * Pause the current animation.
     */
    pause() {
        this.isPaused = true;
    }

    /**
     * Resume the paused animation.
     */
    resume() {
        this.isPaused = false;
    }

    /**
     * Toggle pause/resume state.
     */
    togglePause() {
        this.isPaused = !this.isPaused;
    }

    /**
     * Set animation speed multiplier.
     * @param {number} speed - Speed multiplier (1 = normal, 2 = 2x speed, 0.5 = half speed)
     */
    setSpeed(speed) {
        this.speedMultiplier = Math.max(0.1, Math.min(10, speed));
        this.typeDelay = this.originalTypeDelay / this.speedMultiplier;
        this.lineDelay = this.originalLineDelay / this.speedMultiplier;
    }

    /**
     * Get current animation state.
     */
    getState() {
        return {
            isRunning: this.isRunning,
            isPaused: this.isPaused,
            currentLine: this.currentLineIndex,
            totalLines: this.lines.length,
            speed: this.speedMultiplier
        };
    }

    /**
     * Helper function for animation delays, called with `await`.
     * @param {number} time - Timeout, in ms.
     */
    _wait(time) {
        return new Promise(resolve => setTimeout(resolve, time));
    }

    /**
     * Converts line data objects into line elements.
     *
     * @param {Object[]} lineData - Dynamically loaded lines.
     * @param {Object} line - Line data object.
     * @returns {Element[]} - Array of line elements.
     */
    lineDataToElements(lineData) {
        return lineData.map(line => {
            let div = document.createElement('div');
            div.innerHTML = `<span ${this._attributes(line)}>${line.value || ''}</span>`;
            return div.firstElementChild;
        });
    }

    /**
     * Helper function for generating attributes string.
     *
     * @param {Object} line - Line data object.
     * @returns {string} - String of attributes.
     */
    _attributes(line) {
        let attrs = '';
        Object.keys(line).forEach(function(key) {
            attrs += key === 'type' ? `${this.pfx}="${line[key]}" ` : `${this.pfx}-${key}="${line[key]}" `;
        }, this);

        return attrs;
    }
}

/**
* HTML API: If current script has container(s) specified, initialise Termynal.
*/
if (document.currentScript.hasAttribute('data-termynal-container')) {
    const containers = document.currentScript.getAttribute('data-termynal-container');
    containers.split('|')
        .forEach(container => new Termynal(container))
}