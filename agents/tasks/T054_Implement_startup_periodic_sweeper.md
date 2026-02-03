# Task: T054 - Implement startup + periodic sweeper

## Goal

Implement cleanup of orphan temp dirs and old `.part` files on startup and periodically.

## Required reading (MUST read before starting)

- `docs/file-safety.md`
- `docs/portal-lifecycle.md`

## Deliverables

- Sweeper component with interval config.
- Startup sweep before accepting requests (or soon after).
- Tests or manual steps.

## Steps

1. Scan `.dropserve_tmp` dirs for stale portals/uploads.
2. Delete `.part` older than PART_MAX_AGE if not active.
3. Delete portal temp dir idle older than PORTAL_IDLE_MAX.
4. Log cleanup actions.

## Acceptance criteria

- [ ] Restart after crash cleans old `.part` files.
- [ ] No user final files are deleted.

## Tests to run

- Manual tests B/C in `docs/operations.md`.

## Notes

- None.
