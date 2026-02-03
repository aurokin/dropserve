# Task: T021 - Implement Control API server (local-only)

## Goal

Implement the Control Service listener and its endpoints; ensure it is not meant to be proxied.

## Required reading (MUST read before starting)

- `docs/05_API_Spec.md`
- `docs/03_Architecture.md`
- `README.md`

## Deliverables

- Control HTTP server binding to 127.0.0.1:9090 (configurable).
- `POST /api/control/portals` implementation that creates portals.
- Health endpoint.

## Steps

1. Add configuration for CONTROL_ADDR.
2. Implement create portal: validate dest_abs is an existing directory.
3. Return portal_id and expires_at.
4. Implement GET /api/control/health.

## Acceptance criteria

- [ ] Control service binds only to loopback by default.
- [ ] Create portal fails if dest_abs does not exist or is not a directory.

## Tests to run

- Integration test with local HTTP request to control service.

## Notes

Do not rely on RemoteAddr checks for local-only if this service is ever proxied; it must be separate from the public service.
