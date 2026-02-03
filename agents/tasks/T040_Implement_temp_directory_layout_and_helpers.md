# Task: T040 - Implement temp directory layout and helpers

## Goal

Create helper functions to compute portal temp root and upload temp paths.

## Required reading (MUST read before starting)

- `docs/file-safety.md`
- `docs/architecture.md`

## Deliverables

- Functions to compute: portal_temp_root, upload_part_path, upload_meta_path.
- Ensure temp dir is inside destination directory.

## Steps

1. Add constants/config for TMP_DIR_NAME default `.dropserve_tmp`.
2. Ensure directories are created with safe permissions.
3. Add unit tests for path generation.

## Acceptance criteria

- [ ] Temp paths are always inside DEST.
- [ ] No collisions between portals.

## Tests to run

- Unit tests.

## Notes

- None.
