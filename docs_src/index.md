# TPUSA (a.k.a Kirk) AI — Project overview

This project aims to demonstrate a focused documentation and retrieval assistant specialized for content related to Turning Point USA (TPUSA). The site and codebase include a research dataset collected under `tpusa_crawl/` and precomputed embeddings in `final_embeddings.json` which were produced to support retrieval-augmented tasks such as search, summarization, and question-answering tailored to the collected corpus.

Key points:

- Data provenance: the corpus used for this project comes from publicly available TPUSA pages and related materials stored under `tpusa_crawl/` in this repository. The precomputed vectors are available in `final_embeddings.json` for reproducibility and experimentation.
- Intended capabilities: the specialized AI is intended as a retrieval-augmented assistant for discovery, context-aware summarization, and example-driven code generation tied to the collected materials. It is not intended as an official TPUSA product and the repository is not affiliated with TPUSA.
- Limitations & ethics: the model reflects the content of the source corpus and therefore inherits its biases and perspectives. Before using outputs in public-facing or decision-making contexts, verify claims against primary sources and consider legal and ethical constraints. Do not use the system to create targeted political persuasion; use it for research, archival, or neutral summarization tasks.
- Reproducibility: processing scripts and the data pipeline are organized under `tools/` and `tpusa_crawl/`. See the dedicated page "TPUSA AI" in the documentation for step-by-step notes on reproducing the dataset and embeddings.

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

## Interactive Demo

See kirk-ai in action! The terminal below shows a live demonstration of the key features:

<div id="termynal" data-termynal data-ty-typeDelay="40" data-ty-lineDelay="700">
    <span data-ty="input">./kirk-ai models</span>
    <span data-ty="progress"></span>
    <span data-ty>Available models:</span>
    <span data-ty>✓ llama3.2:3b (chat, coding)</span>
    <span data-ty>✓ gemma2:9b (chat, reasoning)</span>
    <span data-ty>✓ nomic-embed-text (embeddings)</span>
    <span data-ty>✓ qwen2.5-coder:7b (coding specialist)</span>
    <span data-ty></span>
    <span data-ty="input">./kirk-ai chat "What are the key benefits of retrieval-augmented generation?"</span>
    <span data-ty="progress"></span>
    <span data-ty="success">Using model: gemma2:9b</span>
    <span data-ty></span>
    <span data-ty>🤖 RAG combines the best of both worlds:</span>
    <span data-ty></span>
    <span data-ty>• **Knowledge Access**: Retrieves relevant context from your documents</span>
    <span data-ty>• **Accuracy**: Grounds responses in specific source material</span>
    <span data-ty>• **Freshness**: Works with updated content without retraining</span>
    <span data-ty>• **Transparency**: Shows which sources informed the answer</span>
    <span data-ty></span>
    <span data-ty="input">./kirk-ai embed "Machine learning and AI development best practices"</span>
    <span data-ty="info">Using model: nomic-embed-text</span>
    <span data-ty="progress"></span>
    <span data-ty>✅ Generated 768-dimensional embedding vector</span>
    <span data-ty>[0.0423, -0.1892, 0.3441, 0.0891, ...]</span>
    <span data-ty></span>
    <span data-ty="input">./kirk-ai rag "How do I optimize model performance?" --embeddings my_docs.json</span>
    <span data-ty="info">Loaded 1,247 embeddings for RAG</span>
    <span data-ty="info">Using RAG-optimized model: gemma2:9b</span>
    <span data-ty="progress"></span>
    <span data-ty>🎯 **Answer**: Based on your documentation:</span>
    <span data-ty></span>
    <span data-ty>1. **Batch Processing**: Use --batch-size and --concurrency flags</span>
    <span data-ty>2. **Rate Limiting**: Set --rate to avoid API throttling</span>
    <span data-ty>3. **Model Selection**: Use --prefer-fast for lower latency</span>
    <span data-ty>4. **Context Optimization**: Tune --context-size and --similarity-threshold</span>
    <span data-ty></span>
    <span data-ty="success">💡 Performance tip: Use progressive loading with --progressive for large contexts!</span>
</div>

<script>
    document.addEventListener('DOMContentLoaded', function() {
        new Termynal('#termynal');
    });
</script>

## Contributing

- If you add more sources or correct metadata, include provenance (original URL, crawl date) and add tests that verify text processing edge cases.
- Keep datasets and embeddings separated from private keys and avoid committing sensitive credentials.

## Contact & disclaimers

This project is not affiliated with or endorsed by Turning Point USA. If you represent the rights holder for any content included here and want it removed, open an issue or submit a DMCA request following standard GitHub procedures.
