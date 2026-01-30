
# Coding Standards

These standards are intentionally language-agnostic, but tasks may assume Go as the default implementation.

## General

- Keep functions small and testable.
- Prefer explicit error handling over silent fallbacks.
- Never trust client-provided paths.
- Stream uploads directly to disk; avoid loading whole files in memory.
- Every endpoint should log a request id (generate one if missing).

## Logging

- Log portal lifecycle events (create/claim/expire/close).
- Log upload lifecycle (init/start/commit/fail/cleanup).
- Do not log file contents.

## Security posture

- Public service binds to localhost and is proxied by Caddy.
- Caddy enforces LAN-only.
- Control API must be loopback-only and not proxied.

## Go-specific (if using Go)

- Use `context.Context` cancellation for request abort detection.
- Ensure file handles are closed on all error paths.
- Consider fsync behavior as a future enhancement; do not block v1 on it.
