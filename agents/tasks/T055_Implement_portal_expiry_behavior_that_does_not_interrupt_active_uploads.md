# Task: T055 - Implement portal expiry behavior that does not interrupt active uploads

## Goal

Implement expiry timers and closing_requested logic per spec.

## Required reading (MUST read before starting)

- `docs/portal-lifecycle.md`
- `docs/file-safety.md`

## Deliverables

- Timer/cron logic for portal expiry
- Integration tests for expiry mid-transfer semantics

## Steps

1. Mark portals expired only if never used.
2. When open_until passes during IN_USE: set closing_requested but keep accepting ongoing uploads.
3. Close portal when active_uploads == 0.

## Acceptance criteria

- [ ] Portal remains valid mid-transfer after open_until.
- [ ] One-time portal closes after queue completes.

## Tests to run

- Manual test D in `docs/operations.md`.

## Notes

- None.
