# Task: T020 - Implement portal state machine

## Goal

Implement the portal model and state transitions exactly as specified.

## Required reading (MUST read before starting)

- `docs/portal-lifecycle.md`
- `docs/architecture.md`

## Deliverables

- Portal struct/model and in-memory store.
- State transition functions.
- Unit tests for transitions.

## Steps

1. Implement states OPEN/CLAIMED/IN_USE/CLOSING/CLOSED/EXPIRED.
2. Implement transition rules from `docs/portal-lifecycle.md`.
3. Add unit tests for expiry and close semantics.

## Acceptance criteria

- [ ] Unit tests cover main transitions.
- [ ] Portal cannot close while active_uploads > 0.

## Tests to run

- Unit tests: `go test ./...` (or equivalent).

## Notes

- None.
