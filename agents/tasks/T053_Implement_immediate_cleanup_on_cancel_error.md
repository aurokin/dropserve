# Task: T053 - Implement immediate cleanup on cancel/error

## Goal

Ensure PUT handler deletes `.part` and `.json` on any failure or cancellation.

## Required reading (MUST read before starting)

- `docs/file-safety.md`

## Deliverables

- Robust defer/error handling in PUT handler
- Tests or documented manual validation

## Steps

1. Detect request context cancellation.
2. Delete temp files on all non-commit exits.
3. Ensure portal active_uploads decremented.

## Acceptance criteria

- [ ] Canceling a transfer leaves no `.part` in destination temp dir.
- [ ] No partial file appears at final path.

## Tests to run

- Manual test per `docs/operations.md`.

## Notes

- None.
