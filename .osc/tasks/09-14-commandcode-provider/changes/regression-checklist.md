# Regression Checklist

## Automated Checks (2026-09-14)

- PASS: `gofmt -w .`; final `gofmt -l .` produced no paths.
- PASS: `git diff --check`.
- PASS: `./osc gate --cmd "go test ./..."` on final source.
- PASS: `go test -race ./internal/runtime/executor/helps ./internal/runtime/executor ./internal/watcher/synthesizer ./sdk/cliproxy ./sdk/cliproxy/auth -run '^TestCommandCode' -count=1`.
- PASS: `go build -o test-output ./cmd/server && rm test-output`.
- PASS: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/cliproxy-commandcode.Tm64Z9/cli-proxy-api ./cmd/server`.
- PASS: `go test ./internal/runtime/executor/helps -run '^$' -bench '^BenchmarkCommandCodeStream$' -benchmem -benchtime=1s`.

The OSC runner emits pre-existing bash/zsh fzf initialization warnings, but executes
the Go command and reports exit zero. These shell-profile warnings were not changed.

## Covered Cases

- Native NDJSON and SSE, including one-byte UTF-8 fragmentation, CRLF, multiline
  data, unterminated final line, metadata and heartbeat lines.
- Chat/Claude/Responses streaming and non-streaming output; text, reasoning,
  complete and incremental tool calls without duplicate arguments; terminal events.
- Exact Claude tool-linked reasoning restoration, no fabricated signatures.
- Usage publisher provider/executor/auth attribution and input/output/cache counters.
- HTTP and in-stream errors, Retry-After, partial EOF, cancellation and upstream abort.
- Request validation, unsupported operations, event and aggregation bounds.
- Session initialization coalescing, credential isolation, deterministic fingerprints,
  initialization capacity and failure cleanup; per-key catalog caching.
- YAML sanitization, model hash changes and credential removal, exact-case key matching,
  alias reload, local-model mode and two-key round-robin with per-key upstream names.

## Manual Release Checks (Not Run)

- Real account initialization and generation against Command Code.
- Multi-turn tool/analysis continuations in the user's actual clients.
- Image input using a model enabled for the user's account.
- Live quota/cooldown behavior and upstream billing comparison.
- VPS process RSS under representative request sizes and concurrency.
