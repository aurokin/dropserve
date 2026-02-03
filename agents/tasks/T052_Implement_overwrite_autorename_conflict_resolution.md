# Task: T052 - Implement overwrite + autorename conflict resolution

## Goal

Resolve final destination path based on overwrite vs autorename (timestamp) policy.

## Required reading (MUST read before starting)

- `docs/file-safety.md`

## Deliverables

- Function resolve_final_path(dest_abs, relpath, policy) -> final_relpath
- Unit tests for conflict cases

## Steps

1. If overwrite: return original relpath.
2. If autorename and final exists: apply timestamp rename rules.
3. Add suffix _2, _3 if multiple conflicts in same second.

## Acceptance criteria

- [ ] Original filename preserved when no conflict.
- [ ] Timestamp rename used only when necessary.
- [ ] Multiple conflicts handled deterministically.

## Tests to run

- Unit tests.

## Notes

- None.
