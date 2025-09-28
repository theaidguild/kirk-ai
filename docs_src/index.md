# kirk-ai

**kirk-ai** is a compact command-line interface crafted to interact with Ollama AI models. This site provides guided documentation to get you started quickly, explain the architecture, and help you integrate `kirk-ai` into your workflows.


## Quick links

- Installation
- Usage
- Commands
- Architecture
- Contributing

---

## Why kirk-ai?

- Minimal, focused CLI for model interactions
- Clear separation between API client, templates and commands
- Lightweight and easily extensible


### Example — quick chat

```bash
./kirk-ai chat "Hello — what's new?"
```

This repository is designed for contributors and users alike. Use the sidebar to navigate deeper sections or search from the top bar.

## Project goal: a TPUSA-specialized AI

This project aims to demonstrate a focused documentation and retrieval assistant specialized for content related to Turning Point USA (TPUSA). The site and codebase include a research dataset collected under `tpusa_crawl/` and precomputed embeddings in `final_embeddings.json` which were produced to support retrieval-augmented tasks such as search, summarization, and question-answering tailored to the collected corpus.

Key points:

- Data provenance: the corpus used for this project comes from publicly available TPUSA pages and related materials stored under `tpusa_crawl/` in this repository. The precomputed vectors are available in `final_embeddings.json` for reproducibility and experimentation.
- Intended capabilities: the specialized AI is intended as a retrieval-augmented assistant for discovery, context-aware summarization, and example-driven code generation tied to the collected materials. It is not intended as an official TPUSA product and the repository is not affiliated with TPUSA.
- Limitations & ethics: the model reflects the content of the source corpus and therefore inherits its biases and perspectives. Before using outputs in public-facing or decision-making contexts, verify claims against primary sources and consider legal and ethical constraints. Do not use the system to create targeted political persuasion; use it for research, archival, or neutral summarization tasks.
- Reproducibility: processing scripts and the data pipeline are organized under `tools/` and `tpusa_crawl/`. See the dedicated page "TPUSA AI" in the documentation for step-by-step notes on reproducing the dataset and embeddings.
