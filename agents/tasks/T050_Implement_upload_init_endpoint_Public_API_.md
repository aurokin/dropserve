# Task: T050 - Implement upload init endpoint (Public API)

## Goal

Create the endpoint that reserves an upload_id and creates metadata + temp file location.

## Required reading (MUST read before starting)

- `docs/api.md`
- `docs/file-safety.md`
- `docs/file-safety.md`

## Deliverables

- POST /api/portals/{portal_id}/uploads
- Creates `{upload_id}.json` sidecar
- Reserves `.part` path

## Steps

1. Validate client_token.
2. Validate relpath and size.
3. Create sidecar metadata with timestamps and policy.
4. Mark upload status as writing.

## Acceptance criteria

- [ ] Invalid relpath is rejected.
- [ ] Duplicate upload_id committed returns 409.
- [ ] Sidecar metadata exists before PUT begins.

## Tests to run

- Integration tests.

## Notes

- None.
