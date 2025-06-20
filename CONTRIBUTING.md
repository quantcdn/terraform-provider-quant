# Contributing to Terraform Provider Quant

Thank you for your interest in contributing to the QuantCDN Terraform provider! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Code Style](#code-style)

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

- Use the GitHub issue tracker
- Include detailed steps to reproduce the bug
- Include your Terraform configuration (with sensitive data removed)
- Include provider version and Terraform version
- Include any relevant error messages

### Suggesting Enhancements

- Use the GitHub issue tracker
- Describe the enhancement and why it would be useful
- Include examples of how it would be used

### Pull Requests

- Fork the repository
- Create a feature branch
- Make your changes
- Add tests for new functionality
- Ensure all tests pass
- Update documentation
- Submit a pull request

## Development Setup

### Prerequisites

- Go 1.22 or later
- Terraform 1.0 or later
- Git

### Local Development

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/your-username/terraform-provider-quant.git
   cd terraform-provider-quant
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the provider:
   ```bash
   go build -o terraform-provider-quant
   ```

4. Set up development overrides in your Terraform configuration:
   ```hcl
   provider_installation {
     dev_overrides {
       "registry.terraform.io/quantcdn/quant" = "/path/to/terraform-provider-quant"
     }
     direct {}
   }
   ```

## Testing

### Unit Tests

Run unit tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Acceptance Tests

Acceptance tests require a QuantCDN API token and organization. Set up your environment:

```bash
export QUANTCDN_API_TOKEN="your-api-token"
export QUANTCDN_ORGANIZATION="your-organization"
```

Run all acceptance tests:
```bash
TF_ACC=1 go test ./internal/provider/ -v
```

Run specific tests:
```bash
TF_ACC=1 go test ./internal/provider/ -v -run TestAccProjectResource
```

### Test Structure

- Unit tests: `*_test.go` files alongside source code
- Acceptance tests: `*_test.go` files in `internal/provider/`
- Mock tests: Use `httpmock` for API mocking
- Integration tests: Use real API endpoints

## Pull Request Process

1. **Fork and clone** the repository
2. **Create a feature branch** from `main`
3. **Make your changes** following the code style guidelines
4. **Add tests** for new functionality
5. **Update documentation** if needed
6. **Run tests** and ensure they pass
7. **Commit your changes** with clear commit messages
8. **Push to your fork** and create a pull request

### Commit Message Format

Use conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Examples:
- `feat(provider): add base URL configuration support`
- `fix(project): handle null write_token in data source`
- `docs(readme): add contributing guidelines`

### Pull Request Checklist

- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] Branch is up to date with main

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golint` for linting
- Write clear, descriptive comments
- Use meaningful variable and function names

### Terraform Configuration

- Use consistent indentation (2 spaces)
- Use descriptive resource names
- Include comments for complex configurations
- Follow Terraform best practices

### Documentation

- Use clear, concise language
- Include examples for new features
- Update both README.md and docs/ files
- Use proper markdown formatting

## Getting Help

If you need help with your contribution:

- Check existing issues and pull requests
- Ask questions in GitHub issues
- Review the documentation
- Look at existing code examples

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project (MIT License). 