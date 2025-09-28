# TPUSA AI — Project overview

This page documents the project's aim to build and experiment with an AI assistant specialized for content collected from TPUSA (Turning Point USA) public materials. It explains data sources, intended uses, reproducibility notes, and ethical constraints.

## Goals

- Produce a retrieval-augmented assistant that can search, summarize, and answer questions about the TPUSA corpus.
- Provide reproducible tooling and embeddings so others can verify methods and experiment with model choices.
- Document the processing pipeline and metadata for data provenance.

## Data sources & provenance

- Raw pages and artifacts are stored under `tpusa_crawl/` in this repository. The dataset was collected from publicly accessible pages and is included here for research and documentation purposes.
- Precomputed embeddings are stored in `final_embeddings.json`. These were produced by a pipeline that chunks documents, sanitizes HTML/text, and calls an embedding model to produce vector representations for retrieval.

## Intended capabilities

The specialized AI is designed for:

- Retrieval: fast, vector-based search over the corpus using the provided embeddings.
- Summarization: generate concise summaries of individual pages or clusters of pages.
- Contextual Q&A: answer factual questions by retrieving the most relevant passages and generating grounded responses.

## Limitations & responsible use

- The assistant reflects the content and biases present in the source documents. Verify model outputs before presenting them as fact.
- Do not use the assistant to generate targeted political persuasion or microtargeted campaigning. This project is intended for research, archival, and neutral summarization and discovery tasks only.
- Respect copyright and privacy: only use or publish material you have the rights to share, and follow applicable terms of service for the sources.

## Reproducibility & pipeline

- The repository contains tooling under `tools/processor/` and `scripts/` used to crawl, clean, and produce embeddings. Typical steps:
  1. Run the crawler to collect raw HTML (stored under `tpusa_crawl/raw_html/`).
  2. Run the content processor to chunk and clean text.
  3. Produce embeddings for each chunk and store them (the resulting vectors are combined into `final_embeddings.json`).

- See `tools/processor/prepare_embeddings_data.go` and other helper scripts for implementation details. If you want me to add a single-command script or Make target that reproduces the pipeline, I can add it.

## Demo video

<video controls playsinline style="max-width:100%; height:auto; border-radius:8px; box-shadow:0 6px 20px rgba(0,0,0,0.45);">
  <source src="../assets/media/demo.mp4" type="video/mp4">
  Your browser does not support HTML5 video playback. You can download the demo file instead: [Download demo](../assets/media/demo.mp4).
</video>

## Contributing

- If you add more sources or correct metadata, include provenance (original URL, crawl date) and add tests that verify text processing edge cases.
- Keep datasets and embeddings separated from private keys and avoid committing sensitive credentials.

## Contact & disclaimers

This project is not affiliated with or endorsed by Turning Point USA. If you represent the rights holder for any content included here and want it removed, open an issue or submit a DMCA request following standard GitHub procedures.
