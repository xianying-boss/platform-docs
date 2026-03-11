# Active Context

_Update this file at the start and end of every work session._

## Currently Working On
> Just completed: Full Go skeleton v1.0 — all packages, 3 runtimes, 12 tools, Firecracker builder, Docker image

**Next task**: 
1. `go mod tidy && go build ./...` — verify zero compile errors
2. Write `markdown_convert/main.go` and `docx_generate/main.go` WASM sources
3. Write unit tests for `router`, `scheduler/node_selector`, `middleware/ratelimit`

---

## Architecture Decisions (do not change without discussion)

| Decision | Reason |
|----------|--------|
| Fiber v2 instead of Chi | Lower allocations, faster routing, better middleware ergonomics |
| Redis list (BLPOP) instead of Redis Streams | Simpler consumer model, no consumer group management needed at MVP scale |
| Wasmtime CLI instead of wasmtime-go SDK | No CGO dependency → cleaner builds, easier cross-compilation |
| Firecracker via HTTP/Unix socket instead of Go SDK | firecracker-go-sdk has CGO requirements; direct API gives full control |
| VM pool with async replenishment | Minimises allocation latency; failed replenishment logs but doesn't block job |
| DockerRunner for GUI tier | Simplest path to Xvfb + Chromium + LibreOffice in one image |
| Tool I/O via env var + stdout | Universal contract works for Go, Python, Shell, any language |
| Tier injected by handler into Locals | Fixes rate-limit bug — body is parsed once by the handler |

---

## Active Known Issues

### Issue 1 — JWT validation is a stub
**File**: `internal/api/middleware/auth.go` function `validateJWT()`
**Problem**: Skeleton does not verify RS256 signature — accepts any 3-part JWT
**Fix**: Add `github.com/golang-jwt/jwt/v5` to go.mod and implement full RS256 parse

### Issue 2 — WASM tools are Go 1.23 source but build tag requires `js && wasm`
**Files**: `sandbox-tools/wasm/*/main.go`
**Problem**: Standard Go 1.23 WASM target uses `GOOS=wasip1 GOARCH=wasm`, not `js && wasm`
**Fix**: Change build tag to `//go:build wasip1` and compile with `GOOS=wasip1 GOARCH=wasm go build`

### Issue 3 — markdown_convert and docx_generate have no implementation
**Files**: `sandbox-tools/wasm/markdown_convert/` and `docx_generate/`
**Problem**: Only manifests exist, no main.go
**Fix**: Implement markdown_convert using stdlib strings replacement; docx_generate needs an external library

### Issue 4 — VM pool cold-boot in dev mode
**File**: `internal/runtime/microvm/executor.go`
**Problem**: When `KernelPath == ""`, the runtime falls back to direct exec mode
which returns a fake result — suitable for dev but won't be obvious to new contributors
**Fix**: Add a clear log message: "running in microvm dev mode — results are mocked"

---

## Context for Next Session

1. **Compile verification** — run `go mod tidy` first, then `go build ./cmd/api-server ./cmd/node-agent`
2. **WASM build** — WASM tools need `GOOS=wasip1 GOARCH=wasm go build -o tool.wasm .` — update Makefile
3. **Snapshot builder test** — requires a physical Linux machine with `/dev/kvm` accessible
4. **Docker image test** — `docker build -t sandbox/desktop-runner:latest ./docker/desktop-runner`
