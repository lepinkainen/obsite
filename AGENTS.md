# AGENTS.md

See [CLAUDE.md](CLAUDE.md) for full architecture documentation.

## Commands

```bash
go build -o obsite .                      # Build binary
go test ./...                             # Run all tests
go test ./internal/parser -run TestSlugify  # Run single test
go test ./internal/generator -v           # Verbose tests for package
UPDATE_GOLDEN=true go test ./...          # Update golden files
task build                                # Full build (test + lint + build) - ALWAYS run after code changes
```

## Workflow

- **Always run `task build`** as the final step after any code changes to ensure tests pass, linting is clean, and the binary builds successfully.

## Code Style

- **Imports**: stdlib first, blank line, then external deps, blank line, then internal (`obsite/internal/...`)
- **Errors**: Return errors up the stack, don't log; use `fmt.Errorf("context: %w", err)` for wrapping
- **Tests**: Table-driven tests preferred; golden files in `testdata/` (`.input.md` → `.golden.md`)
- **Naming**: Exported functions use PascalCase; keep receiver names short (e.g., `p` for `*Post`)
- **No comments** on obvious code; doc comments on exported functions only
