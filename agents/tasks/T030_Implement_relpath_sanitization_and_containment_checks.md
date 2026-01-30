# Task: T030 - Implement relpath sanitization and containment checks

## Goal

Build a path normalization library and tests to prevent traversal and unsafe paths.

## Required reading (MUST read before starting)

- `docs/07_Path_Safety.md`
- `docs/06_File_IO_and_Cleanup.md`

## Deliverables

- sanitize_relpath(input) -> cleaned_relpath or error.
- join_and_verify(dest_abs, cleaned_relpath) -> final_abs or error.
- Unit tests from `docs/07_Path_Safety.md`.

## Steps

1. Implement all rejection rules.
2. Ensure Windows drive letters and absolute paths are rejected.
3. Add tests for accept/reject cases.

## Acceptance criteria

- [ ] All test cases in `docs/07_Path_Safety.md` pass.
- [ ] No path traversal possible.

## Tests to run

- Unit tests: `go test ./...` (or equivalent).

## Notes

- None.
