# Contributing to mydevstack-proxy

Thank you for your interest in contributing!

## Getting Started

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/YOUR_USERNAME/mydevstack-proxy.git`
3. **Create** a feature branch: `git checkout -b feature/my-new-feature`

## Development Setup

```bash
# Install dependencies
go mod download

# Run the project
go run main.go

# Run tests
go test -v ./...

# Build
go build -o mydevstack-proxy .
```

## Code Standards

- Follow Go best practices and idiomatic patterns
- Use **type-first development** - define types and interfaces before implementation
- **Accept interfaces, return structs** - define minimal interfaces in consumer packages
- Use **small interfaces** (1-2 methods) where possible
- Add **compile-time interface assertions** in `internal/adapters/aws/assert.go`

### Architecture Guidelines

This project follows **Hexagonal Architecture** (Ports and Adapters):

```
main.go → Handlers → Application Service → AWS Adapters
                ↓              ↓                    ↓
           Ports (Interfaces)                      AWS SDK
```

- **Ports** (`internal/ports/`) - Define interfaces for external dependencies
- **Adapters** (`internal/adapters/`) - Implementations that connect to external systems
- **Application** (`internal/application/`) - Business logic and orchestration

## Submitting Changes

1. **Test** your changes thoroughly
2. **Format** your code: `go fmt ./...`
3. **Lint** your code: `golangci-lint run`
4. **Commit** with clear, descriptive messages
5. **Push** to your fork
6. **Open** a Pull Request against `main` branch

### Pull Request Guidelines

- Describe the changes and the motivation
- Link to any related issues
- Ensure all tests pass
- Include any documentation updates

## Reporting Bugs

1. Check if the issue already exists
2. Create a detailed issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Go version and environment details

## Questions?

Feel free to open an issue for questions about contributing or the project in general.