# Contributing

Thanks for considering contributing! Guidelines:

- Run `go fmt ./...` before committing.
- Run `go vet ./...` to catch vet issues.
- Add tests alongside code in `_test.go` files.
- For documentation changes, edit the markdown files in `docs_src/` and preview locally with `mkdocs serve`.

To publish docs changes:

1. Edit files in `docs_src/` and commit to `main`.
2. Push the commit to `main` and the Pages workflow will build and deploy the site.
