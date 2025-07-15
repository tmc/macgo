# Contributing to macgo

Thank you for your interest in contributing to macgo! This document provides guidelines and instructions for contributing to the project.

## Development Setup

### Prerequisites

- macOS (required for testing)
- Go 1.22 or later
- Git
- Make (optional but recommended)

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/macgo.git
   cd macgo
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/tmc/misc.git
   ```

4. Install development tools:
   ```bash
   make install-lint-tools
   make install-security-tools
   make install-hooks  # Installs pre-commit hooks
   ```

## Development Workflow

### Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the code style guidelines

3. Format your code:
   ```bash
   make fmt
   ```

4. Run tests:
   ```bash
   make test
   make integration-test  # For integration tests
   ```

5. Run linters:
   ```bash
   make lint
   ```

6. Commit your changes:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

### Commit Message Format

Follow conventional commits format:
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Test additions or changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks
- `ci:` CI/CD changes

### Testing Requirements

All contributions must include appropriate tests:

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **Example Programs**: Update or add examples if adding new features

Run all tests before submitting:
```bash
make all  # Runs fmt, vet, lint, test, and build
```

### Code Review Process

1. Submit a pull request against the main branch
2. Ensure all CI checks pass
3. Respond to reviewer feedback
4. Once approved, the PR will be merged

## Testing Guidelines

### macOS-Specific Testing

Since macgo is macOS-specific, some tests require special handling:

1. **Sandbox Tests**: Cannot be fully automated in CI
   - Mock permission checks where possible
   - Document manual testing requirements

2. **Signal Handling**: Use timeouts to prevent hanging
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   ```

3. **App Bundle Tests**: Clean up created bundles
   ```go
   defer os.RemoveAll(bundlePath)
   ```

### Writing Tests

Example test structure:
```go
func TestFeature(t *testing.T) {
    // Setup
    originalValue := someGlobalValue
    defer func() { someGlobalValue = originalValue }()
    
    // Test cases
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "expected", false},
        {"invalid input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
            }
            if result != tt.expected {
                t.Errorf("Feature() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## CI/CD Integration

The project uses GitHub Actions for CI/CD. All PRs must pass:

1. **Linting**: Code formatting and style checks
2. **Unit Tests**: On multiple macOS and Go versions
3. **Integration Tests**: macOS-specific functionality
4. **Security Scans**: Vulnerability detection
5. **Example Verification**: All examples must compile

## Security

### Reporting Security Issues

Please report security vulnerabilities privately to the maintainers. Do not open public issues for security problems.

### Security Checks

Before submitting code:
```bash
make security-scan
```

Address any findings before submitting your PR.

## Documentation

### Code Documentation

- Document all exported types, functions, and methods
- Use clear, concise language
- Include examples where helpful

### Example Documentation
```go
// CreateBundle creates a macOS application bundle for the given executable.
// It returns the path to the created bundle and any error encountered.
//
// Example:
//   bundlePath, err := CreateBundle("/path/to/executable")
//   if err != nil {
//       log.Fatal(err)
//   }
func CreateBundle(execPath string) (string, error) {
    // Implementation
}
```

## Questions and Support

- Open an issue for bugs or feature requests
- Use discussions for questions and ideas
- Check existing issues before creating new ones

## License

By contributing to macgo, you agree that your contributions will be licensed under the same license as the project.