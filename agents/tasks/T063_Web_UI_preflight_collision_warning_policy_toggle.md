# Task: T063 - Web UI: preflight collision warning + policy toggle

## Goal

Add preflight call and UI warnings for overwrites with autorename option.

## Required reading (MUST read before starting)

- `docs/web-ui.md`
- `docs/api.md`

## Deliverables

- Preflight request + UI
- Policy selection stored for subsequent init calls

## Steps

1. Before first upload, call preflight with queue relpaths.
2. If conflicts, display warning and offer toggle autorename.
3. Store policy per queue session.

## Acceptance criteria

- [ ] Conflicts are shown before uploading.
- [ ] Autorename policy is applied to uploads.

## Tests to run

- Manual test E in `docs/operations.md`.

## Notes

- None.
