# Task: T070 - CLI: implement primary LAN IPv4 detection

## Goal

Implement IP detection that produces a usable LAN link by default.

## Required reading (MUST read before starting)

- `docs/09_CLI_Spec.md`

## Deliverables

- Function detect_primary_ipv4()
- Documented behavior for edge cases

## Steps

1. Implement UDP connect method.
2. Fallback to interface enumeration.
3. Add --host override support.

## Acceptance criteria

- [ ] Returns expected LAN IPv4 in typical Ubuntu server network.
- [ ] Does not return 127.0.0.1 unless no other option.

## Tests to run

- Manual test on server.

## Notes

- None.
