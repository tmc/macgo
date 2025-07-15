# CI/CD Pipeline Documentation for macgo

## Overview

This document describes the comprehensive CI/CD pipeline implemented for the macgo project. The pipeline is designed to ensure code quality, security, and compatibility across different macOS versions while maintaining the library's reliability as an open-source macOS development tool.

## Pipeline Architecture

### GitHub Actions Workflows

The CI/CD pipeline consists of several GitHub Actions workflows:

1. **CI Workflow** (`ci.yml`) - Main continuous integration pipeline
2. **Release Workflow** (`release.yml`) - Automated release process
3. **CodeQL Analysis** (`codeql.yml`) - Security and code quality analysis
4. **Dependency Review** (`dependency-review.yml`) - Dependency security scanning
5. **macOS Compatibility Testing** (`macOS-compatibility.yml`) - Cross-version testing

### Automation Tools

- **Dependabot** - Automated dependency updates
- **Makefile** - Local development and CI coordination

## Workflow Details

### 1. Continuous Integration (CI)

The main CI workflow runs on every push and pull request to the main/master branch.

#### Jobs:

**Lint Job**
- Runs on: `macos-latest`
- Checks:
  - Go formatting (`gofmt -s`)
  - Go vet analysis
  - Staticcheck for advanced static analysis
  - Golint for style compliance

**Test Job**
- Matrix testing across:
  - macOS versions: 13, 14
  - Go versions: 1.22, 1.23, 1.24
- Features:
  - Race condition detection (`-race` flag)
  - Code coverage reporting
  - Atomic coverage mode for accuracy
  - Upload to Codecov for tracking

**Integration Test Job**
- Runs macOS-specific integration tests
- Tests signal handling comprehensively
- Verifies bundle creation functionality
- Uses timeouts to prevent hanging tests

**Example Verification Job**
- Builds all examples to ensure they compile
- Runs basic examples with process management
- Validates that examples remain functional

**Security Scan Job**
- Runs gosec for security vulnerability detection
- Performs dependency vulnerability scanning
- Uploads results in SARIF format for GitHub integration

**Benchmark Job**
- Runs performance benchmarks
- Captures memory allocation statistics
- Archives results for performance tracking

**Build Artifacts Job**
- Builds release binaries
- Uploads artifacts for distribution

### 2. Release Workflow

Automated release process triggered by version tags or manual dispatch.

**Features:**
- Version validation (semantic versioning)
- Multi-architecture builds (AMD64, ARM64)
- Universal binary creation using `lipo`
- Automated changelog generation
- GitHub Release creation with assets
- Documentation publishing

**Process:**
1. Validate version format
2. Run full test suite
3. Build for multiple architectures
4. Create universal binary
5. Generate changelog from git history
6. Create GitHub Release with artifacts
7. Publish documentation updates

### 3. Security Scanning

**CodeQL Analysis**
- Runs on push, PR, and weekly schedule
- Languages: Go
- Query suites: security-extended, security-and-quality
- Uploads results to GitHub Security tab

**Dependency Review**
- Runs on all pull requests
- Checks for:
  - Known vulnerabilities
  - License compliance
  - Severity thresholds (fails on moderate+)

### 4. macOS Compatibility Testing

Comprehensive testing across macOS versions and architectures.

**Test Matrix:**
- macOS Sonoma 14.x (Apple Silicon/ARM64)
- macOS Ventura 13.x (Intel/x86_64)
- macOS Monterey 12.x (Intel/x86_64)

**Tests Include:**
- System information verification
- Architecture-specific builds
- App bundle creation
- Entitlements functionality
- Signal handling
- Sandbox compatibility
- Framework availability
- Cross-compilation verification

### 5. Dependency Management

**Dependabot Configuration:**
- Go modules: Weekly updates on Monday
- GitHub Actions: Weekly updates
- Pull request limits to prevent spam
- Automatic labeling and commit prefixes

## Local Development Support

### Makefile Targets

The project includes a comprehensive Makefile for local development:

```bash
make all              # Run fmt, vet, lint, test, and build
make test             # Run unit tests
make integration-test # Run integration tests
make coverage         # Generate test coverage report
make benchmark        # Run benchmarks
make build            # Build all binaries
make clean            # Clean build artifacts
make fmt              # Format code
make vet              # Run go vet
make lint             # Run linters
make security-scan    # Run security scans
make verify-examples  # Verify all examples compile
make pre-commit       # Run pre-commit checks
make install-hooks    # Install git pre-commit hooks
```

## Quality Gates

### Code Coverage
- Minimum coverage threshold: Monitored via Codecov
- Coverage reports generated for each OS/Go version combination
- Atomic coverage mode for accurate concurrent test coverage

### Security Requirements
- No high or critical vulnerabilities in dependencies
- Security scanning must pass before merge
- SARIF reports integrated with GitHub Security

### Testing Requirements
- All tests must pass on all supported macOS versions
- Integration tests must complete within timeout
- Examples must compile successfully
- Race condition detection enabled

## macOS-Specific Considerations

### Testing Challenges

1. **Sandbox Testing**
   - Cannot fully test sandbox functionality in CI
   - Mock testing for permission checks
   - Build verification only for sandboxed examples

2. **TCC Permissions**
   - Cannot grant permissions in CI environment
   - Test permission checking code only
   - Document manual testing requirements

3. **Signal Handling**
   - Use timeouts to prevent hanging tests
   - Test signal forwarding with process management
   - Verify cleanup on test completion

4. **App Bundle Creation**
   - Test bundle structure creation
   - Cannot test code signing in CI
   - Verify entitlements file generation

### Environment Variables

The pipeline uses several environment variables:

- `MACGO_TEST_INTEGRATION=1` - Enable integration tests
- `MACGO_DEBUG=1` - Enable debug logging
- `CGO_ENABLED=1` - Required for macOS system integration

## Maintenance Guidelines

### Adding New Tests

1. Place unit tests alongside source files
2. Use build tags for integration tests: `//go:build integration`
3. Add timeouts for tests that spawn processes
4. Clean up resources in test cleanup functions

### Updating CI Configuration

1. Test workflow changes in a feature branch
2. Use matrix strategy for version testing
3. Pin action versions for stability
4. Document any new environment requirements

### Release Process

1. Update version in code if needed
2. Create and push version tag: `git tag v1.2.3`
3. Monitor release workflow execution
4. Verify published artifacts
5. Update documentation if needed

### Security Updates

1. Review Dependabot PRs promptly
2. Run security scans locally before major changes
3. Address vulnerabilities based on severity
4. Document security-related changes

## Monitoring and Alerts

### Recommended Monitoring

1. **Build Status**
   - Monitor workflow success rates
   - Track build times for performance regression
   - Alert on repeated failures

2. **Test Coverage**
   - Track coverage trends over time
   - Alert on significant coverage drops
   - Review uncovered code in PRs

3. **Security Scanning**
   - Review CodeQL alerts regularly
   - Monitor dependency vulnerabilities
   - Track time to remediation

4. **Release Health**
   - Monitor download statistics
   - Track issue reports post-release
   - Verify artifact integrity

## Troubleshooting

### Common Issues

1. **Test Timeouts**
   - Increase timeout values for slow tests
   - Add progress logging for long operations
   - Use goroutine dumps for deadlock debugging

2. **macOS Version Compatibility**
   - Check for deprecated APIs
   - Test on minimum supported version
   - Document version-specific features

3. **Permission Failures**
   - Cannot test actual permission grants in CI
   - Focus on permission checking logic
   - Document manual testing requirements

4. **Signal Handling Issues**
   - Use process groups for cleanup
   - Implement proper signal forwarding
   - Add diagnostic logging for debugging

## Future Enhancements

### Planned Improvements

1. **Performance Tracking**
   - Automated benchmark regression detection
   - Performance dashboard integration
   - Historical trend analysis

2. **Extended Compatibility**
   - Beta macOS version testing
   - Xcode version matrix
   - Swift integration testing

3. **Advanced Security**
   - SAST tool integration
   - Container scanning for examples
   - Supply chain security (SLSA)

4. **Documentation Automation**
   - API documentation generation
   - Example validation in docs
   - Automated changelog updates