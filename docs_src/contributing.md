# Contributing

Thanks for contributing! We welcome fixes, documentation improvements, and new features.

## Development workflow

- Fork the repo and create a feature branch.
- Run `go fmt ./...` and `go vet ./...` before committing.
- Add tests alongside your code changes.

## Docs contributions

- Edit markdown files under `docs_src/`.
- Preview locally with `mkdocs serve -f mkdocs.yml`.
- Open a PR against `main` — the Pages CI pipeline will build and deploy the docs automatically.

## Reviewing

Keep changes small and focused. For large changes, open an issue first to discuss the approach.
