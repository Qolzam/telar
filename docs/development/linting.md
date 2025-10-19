# Linting Guide

Go code quality is enforced using `golangci-lint` with 20+ standard linters and custom analyzers.

## Setup

**Install golangci-lint:**
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Verify
golangci-lint --version
```

## Usage

```bash
# Run all linters
make lint

# Auto-fix issues
make lint-fix

# Run on specific package
golangci-lint run ./posts/...
```

## Configuration

**Location:** `.golangci.yml` (root level)

**Enabled Linters:**
- Code quality: `gofmt`, `goimports`, `govet`, `gosimple`, `staticcheck`
- Error handling: `errcheck`, `ineffassign`
- Security: `gosec`
- Complexity: `gocyclo`, `gocognit`, `funlen`
- Best practices: `goconst`, `mnd`, `prealloc`, `nakedret`, `dupl`
- Style: `gocritic`, `misspell`, `unused`

**Editor Configuration:** `.editorconfig` ensures consistent code style across editors.

## Custom Linters

**Location:** `tools/linters/`

### no_setenv_in_tests
Prevents `os.Setenv()` and `t.Setenv()` in test files to enforce Config-First dependency injection and prevent data races.

```go
// ❌ Bad
func TestFeature(t *testing.T) {
    os.Setenv("DB_URL", "test")
}

// ✅ Good
func TestFeature(t *testing.T) {
    cfg := *testutil.DefaultConfig()
    cfg.DatabaseURL = "test"
    service := NewService(&cfg)
}
```

**Build custom linters:**
```bash
cd tools/linters
go build -o no_setenv_in_tests .
```

## Common Issues

### Unchecked Errors
```go
// ❌ Bad
file.Close()

// ✅ Good
if err := file.Close(); err != nil {
    return fmt.Errorf("close file: %w", err)
}
```

### Magic Numbers
```go
// ❌ Bad
if len(items) > 5 {

// ✅ Good
const maxItems = 5
if len(items) > maxItems {
```

### Disable Linter (sparingly)
```go
//nolint:gosec // test password
password := "test123"
```

## CI Integration

Linters run automatically via `make ci-fast` and `make ci-test`.

## References

- [golangci-lint docs](https://golangci-lint.run/)
- Custom linters: `tools/linters/README.md`
