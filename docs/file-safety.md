# File Safety and Cleanup

This document is the source of truth for file safety, cleanup, and path handling.

## Core invariants

1. No partial uploads ever appear in the final destination.
2. Uploads stream to disk without buffering full files in memory.
3. Final files appear only after successful verification and atomic rename.
4. Failed uploads delete temp artifacts immediately.
5. Startup and periodic sweeps clean orphaned temp artifacts.

## Temp layout

For destination `DEST` and portal `P`:

- Portal temp root: `DEST/.dropserve_tmp/P/`
- Upload temp file: `DEST/.dropserve_tmp/P/uploads/{upload_id}.part`
- Upload metadata: `DEST/.dropserve_tmp/P/uploads/{upload_id}.json`

The temp layout lives inside `DEST` to allow atomic rename on commit.

## Upload algorithm (per file)

1. Init: create portal temp root and `{upload_id}.json` metadata.
2. PUT stream: write to `{upload_id}.part`, track bytes + SHA-256.
3. On stream error: delete `.part` and `.json`, mark failed.
4. Verify: bytes match expected size; optional client hash matches.
5. Resolve final relpath (overwrite or autorename).
6. Commit: create parent dirs, atomic rename to final path, delete `.json`.

## Auto-rename rule

When a conflict exists, rename `name.ext` to:

- `name_YYYY-MM-DD_HHMMSS.ext`
- If still exists, add `_2`, `_3`, etc.

Keep the original name whenever possible.

## Cleanup strategy

- **Immediate**: delete `.part` and `.json` on failure or cancel.
- **On close/expire**: delete `DEST/.dropserve_tmp/P/`.
- **Sweeper**: on startup and periodically, remove stale temp artifacts.

Recommended defaults:
- Sweep interval: 2 minutes.
- Delete `.part` older than: 10 minutes (if not active).
- Delete portal temp dirs idle older than: 30 minutes.

## Path safety rules

All `relpath` values are untrusted. Apply these rules:

1. Replace `\\` with `/`.
2. Reject empty paths, NUL, absolute paths, `~/`, Windows drive prefixes, or any `..` segment.
3. Clean with POSIX rules (`a//b` -> `a/b`, `a/./b` -> `a/b`).
4. Join with `dest_abs` and verify the final path is contained within `dest_abs`.

### Must-reject examples

- `../etc/passwd`
- `a/../../b`
- `/absolute/path`
- `C:\\Windows\\System32`
- `a/../b`

### Must-accept examples

- `a/b/c.txt`
- `a//b///c.txt` -> `a/b/c.txt`
- `a/./b/c.txt` -> `a/b/c.txt`
