# Chitragupt

Go CLI that converts CLI agent session logs into shareable transcripts.

## Testing

- Use `testdata/*.jsonl` files for synthetic test fixtures, not inline string builders
- Use `github.com/stretchr/testify` (`require` for fatal checks, `assert` for the rest)
- Table-driven tests with `t.Run` subtests
- For tests that need directory structure (e.g. Reader methods that traverse dirs), copy testdata files into `t.TempDir()`

## Binaries

- Always build with `make build`
- Build binaries must be in `.bin/`
