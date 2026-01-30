# Task: T022 - Implement Public API: claim portal

## Goal

Implement portal claim behavior and client_token enforcement for one-time portals.

## Required reading (MUST read before starting)

- `docs/05_API_Spec.md`
- `docs/04_Portal_Spec.md`

## Deliverables

- POST /api/portals/{portal_id}/claim endpoint.
- Middleware/helper for X-Client-Token validation.

## Steps

1. Claim returns client_token and policy info.
2. For one-time portals, reject second claim with 409.
3. All state-changing endpoints must validate X-Client-Token.

## Acceptance criteria

- [ ] Claim semantics match `docs/04_Portal_Spec.md`.
- [ ] Token required for upload init, upload PUT, close.

## Tests to run

- Unit/integration tests for claim and token enforcement.

## Notes

- None.
