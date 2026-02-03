# Task: T051 - Implement upload PUT streaming endpoint (Public API)

## Goal

Stream request body to temp file while computing SHA-256 and counting bytes.

## Required reading (MUST read before starting)

- `docs/api.md`
- `docs/file-safety.md`

## Deliverables

- PUT /api/uploads/{upload_id}
- Server computes sha-256 during streaming
- Byte count verification

## Steps

1. Open `.part` for write.
2. Stream body to disk with constant memory.
3. On EOF: verify bytes == expected size.
4. If client_sha256 exists: compare.
5. Commit via atomic rename.
6. Delete sidecar `.json`.

## Acceptance criteria

- [ ] Large file uploads do not consume large RAM.
- [ ] Size mismatch fails and deletes temp artifacts.
- [ ] Committed upload returns server_sha256.

## Tests to run

- Integration tests with a multi-MB file.
- Cancel test using client disconnect (if feasible).

## Notes

- None.
