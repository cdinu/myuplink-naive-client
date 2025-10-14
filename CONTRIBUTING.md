# Contributing

Thanks for your interest in improving the MyUplink Naive Client!

## Development Environment

1. Install Go 1.22 or newer.
2. Install development tools:
   ```sh
   go install mvdan.cc/gofumpt@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```
3. Fork and clone the repository, then create a feature branch.

## Workflow

- Make small, focused changes.
- Run the full suite before opening a pull request:
  ```sh
  make fmt
  make lint
  make test
  ```
- Add or update tests whenever behaviour changes.
- Write clear commit messages that explain the reasoning behind the change.

## Pull Requests

- Ensure `go test ./...` passes and the linter reports no issues.
- Include updates to documentation or examples when behaviour changes.
- Describe testing performed and any follow-up work.

Thank you for helping keep the project reliable!
