# Custom Linters for Telar

This directory contains custom Go linters that enforce Telar-specific coding patterns and best practices.

## Available Linters

### `no_setenv_in_tests`

Prevents the use of `os.Setenv` and `t.Setenv` in test files to enforce the Config-First dependency injection pattern.

**Why?**
- `os.Setenv` and `t.Setenv` modify global state
- They can cause data races in parallel test execution
- They violate the Config-First pattern

**Instead, do this:**
```go
func TestMyFeature(t *testing.T) {
    // 1. Get base config from testutil
    cfg, cleanup := testutil.Setup(t)
    defer cleanup()
    
    // 2. Create a local copy and modify it
    testCfg := *cfg
    testCfg.SomeSetting = "custom-value"
    
    // 3. Pass the modified config to constructors
    service := NewService(&testCfg)
    
    // Test your feature
}
```

## Building the Linters

```bash
cd tools/linters
go mod download
go build -o no_setenv_in_tests .
```

## Integration with golangci-lint

The custom linters are automatically integrated via the `.golangci.yml` configuration at the project root:

```yaml
custom:
  no_setenv_in_tests:
    path: "./tools/linters"
    description: "Prevent os.Setenv and t.Setenv usage in test files"
```

## Running the Linters

Use the Makefile targets:

```bash
# Run all linters
make lint

# Run linters with auto-fix
make lint-fix
```

## Adding New Custom Linters

1. Create a new analyzer in this directory
2. Add it to the `Analyzer` variable in `main.go`
3. Update the `.golangci.yml` configuration
4. Document it in this README
