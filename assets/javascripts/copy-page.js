// Adds a "Copy page URL" button to the top of each docs page
document.addEventListener('DOMContentLoaded', function () {
  try {
    var header = document.querySelector('.md-header__inner');
    if (!header) return;

    var btn = document.createElement('button');
    btn.className = 'md-btn md-btn--secondary md-icon md-icon--copy';
    btn.title = 'Copy page URL';
    btn.innerText = 'Copy page';
    btn.style.marginLeft = '12px';

    btn.addEventListener('click', function () {
      var url = window.location.href;
      navigator.clipboard.writeText(url).then(function () {
        btn.innerText = 'Copied';
        setTimeout(function () { btn.innerText = 'Copy page'; }, 1200);
      }).catch(function () {
        btn.innerText = 'Copy URL';
      });
    });

    // Insert the button near the header actions area
    var actions = header.querySelector('.md-header__title') || header;
    actions.appendChild(btn);
  } catch (e) {
    console.warn('copy-page script failed', e);
  }
});