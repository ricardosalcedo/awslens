# Contributing to awslens

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

```bash
git clone https://github.com/ricardosalcedo/awslens.git
cd awslens
go build -o awslens .
```

Requires Go 1.21+ and AWS credentials configured locally.

## Making Changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Run `go vet ./...` and `go test ./...`
4. Commit with a clear message describing the change
5. Open a pull request

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small
- Add comments for exported types and functions

## Reporting Bugs

Open an issue with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Your OS and Go version

## Feature Requests

Open an issue describing the feature and why it would be useful. For large changes, please discuss in an issue before starting work.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
