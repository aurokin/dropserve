# Task: T071 - CLI: open portal command

## Goal

Implement `dropserve open` that calls Control API and prints the portal link.

## Required reading (MUST read before starting)

- `docs/09_CLI_Spec.md`
- `docs/05_API_Spec.md`
- `docs/04_Portal_Spec.md`

## Deliverables

- CLI command with flags --minutes, --reusable, --policy
- Calls control endpoint and prints URL

## Steps

1. Canonicalize current directory to dest_abs.
2. POST to control create portal endpoint.
3. Detect LAN IPv4.
4. Print http://{ip}/p/{portal_id} (or include port if non-80).

## Acceptance criteria

- [ ] Portal is created successfully and link works from another LAN machine.
- [ ] Flags correctly change duration and reusability.

## Tests to run

- Manual test.

## Notes

- None.
