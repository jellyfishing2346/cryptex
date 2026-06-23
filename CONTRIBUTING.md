# Contributing to Cryptex

Thank you for your interest in contributing to Cryptex! This document provides guidelines and instructions for contributing to the project.

## 🤝 How to Contribute

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title**: Describe the bug briefly
- **Description**: Detailed explanation of the problem
- **Steps to reproduce**: Step-by-step instructions to reproduce the issue
- **Expected behavior**: What you expected to happen
- **Actual behavior**: What actually happened
- **Environment**:
  - OS and version
  - Go version
  - Redis version
  - Cryptex version or commit hash
- **Logs**: Relevant error messages or logs
- **Additional context**: Any other relevant information

### Suggesting Enhancements

Enhancement suggestions are welcome! Please include:

- **Clear title**: Brief description of the enhancement
- **Problem statement**: What problem this enhancement would solve
- **Proposed solution**: How you envision the enhancement working
- **Alternatives considered**: Other approaches you've considered
- **Additional context**: Any other relevant information

### Pull Requests

We welcome pull requests! Here's how to contribute:

1. **Fork the repository**
   ```bash
   # Fork the repository on GitHub
   # Clone your fork
   git clone https://github.com/YOUR_USERNAME/cryptex.git
   cd cryptex
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

3. **Make your changes**
   - Follow the code style guidelines
   - Add tests for new functionality
   - Update documentation as needed
   - Ensure all tests pass

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request**
   - Go to the original repository on GitHub
   - Click "New Pull Request"
   - Select your branch
   - Fill in the PR template
   - Submit for review

## 📝 Development Guidelines

### Code Style

We follow standard Go conventions:

- **Formatting**: Use `gofmt` for all code
  ```bash
  go fmt ./...
  ```

- **Linting**: Use `golangci-lint`
  ```bash
  golangci-lint run
  ```

- **Naming**:
  - Packages: short, lowercase, single words
  - Exports: PascalCase
  - Private: camelCase
  - Constants: PascalCase (exported), camelCase (private)

- **Comments**:
  - Package comments for each package
  - Exported functions must have comments
  - Complex logic should be explained
  - Use TODO comments for future work

### Testing Requirements

- **Unit tests**: Required for all new functions
- **Integration tests**: Required for API endpoints
- **Coverage**: Aim for >85% coverage on new code
- **Benchmarks**: Add for performance-critical code

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/matching/...

# Run benchmarks
go test -bench=. ./internal/orderbook/
```

### Documentation Requirements

- **README**: Update if features change
- **API docs**: Update if API endpoints change
- **Architecture**: Update if system design changes
- **Comments**: Document complex logic
- **Examples**: Provide usage examples for new features

## 🏗️ Project Structure

```
cryptex/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── matching/       # Order matching engine
│   ├── models/         # Data models
│   ├── nats/           # NATS publisher
│   ├── orderbook/      # Order book implementation
│   ├── persistence/    # Redis storage
│   ├── risk/           # Risk management
│   └── ws/             # WebSocket handler
├── web/                # Web dashboard
├── docs/               # Documentation
├── docker/             # Docker configuration
└── deploy/             # Deployment manifests
```

## 📋 Commit Message Convention

We follow conventional commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test additions/changes
- `chore`: Build process or auxiliary tool changes
- `ci`: CI/CD changes

### Examples

```
feat(matching): add stop-limit order support

Add support for stop-limit orders with trigger price
and limit price validation.

Closes #123
```

```
fix(api): resolve race condition in order book

Fix race condition when multiple orders are added
simultaneously to the same price level.

Fixes #456
```

```
docs(readme): update installation instructions

Add Docker installation instructions and update
Go version requirements.
```

## 🧪 Testing Strategy

### Unit Tests

Test individual functions and methods in isolation:

```go
func TestOrderBookAdd(t *testing.T) {
    book := orderbook.New("BTC-USD")
    order := &models.Order{
        ID:       uuid.New(),
        Side:     models.SideBuy,
        Price:    50000.0,
        Quantity: 1.0,
    }

    err := book.Add(order)
    if err != nil {
        t.Fatalf("Add() error = %v", err)
    }

    _, exists := book.GetOrder(order.ID)
    if !exists {
        t.Error("Order not found in book")
    }
}
```

### Integration Tests

Test component interactions:

```go
func TestAPIPlaceOrderIntegration(t *testing.T) {
    book := orderbook.New("BTC-USD")
    engine := matching.New(book)
    server := api.NewServer(book, engine, nil)

    // Make HTTP request
    // Verify response
    // Check order book state
}
```

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestEngineSubmit(t *testing.T) {
    tests := []struct {
        name     string
        order    *models.Order
        setup    func(*orderbook.OrderBook)
        wantErr  bool
        validate func(*Result, error)
    }{
        {
            name: "market order fills completely",
            order: &models.Order{
                Side:     models.SideBuy,
                Type:     models.OrderTypeMarket,
                Quantity: 1.0,
            },
            wantErr: false,
            validate: func(result *Result, err error) {
                // Validate result
            },
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## 🔍 Code Review Process

### Before Submitting

- [ ] Code follows style guidelines
- [ ] All tests pass
- [ ] New tests added for new features
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] No unnecessary changes
- [ ] No merge conflicts

### During Review

- **Be constructive**: Provide helpful feedback
- **Be specific**: Point to exact lines/issues
- **Be respectful**: Maintain professional tone
- **Ask questions**: Clarify intent if unclear
- **Suggest improvements**: Offer better alternatives

### After Review

- **Address feedback**: Make requested changes
- **Discuss alternatives**: If you disagree, explain why
- **Update PR**: Mark conversations as resolved
- **Re-request review**: When changes are complete

## 🚀 Release Process

### Versioning

We follow semantic versioning (MAJOR.MINOR.PATCH):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

### Release Checklist

- [ ] Update version in code
- [ ] Update CHANGELOG.md
- [ ] Tag the release
- [ ] Create GitHub release
- [ ] Update documentation
- [ ] Test release artifacts

### Creating a Release

```bash
# Update version
# Update CHANGELOG.md

# Commit changes
git add .
git commit -m "chore: release v1.0.0"

# Create tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push
git push origin main
git push origin v1.0.0
```

## 🌟 Recognition

Contributors are recognized in:

- CONTRIBUTORS.md file
- Release notes
- GitHub contributors list

## 📞 Getting Help

If you need help contributing:

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and ideas
- **Documentation**: Check existing docs first
- **Code Examples**: Look at existing code for patterns

## 📜 License

By contributing, you agree that your contributions will be licensed under the MIT License.

## 🙏 Thank You

Thank you for contributing to Cryptex! Your contributions help make this project better for everyone.
