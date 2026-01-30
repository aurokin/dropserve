
# Path Safety Specification

This document defines how to safely handle user-supplied relative paths from browsers.

## Inputs

Browsers supply `relpath` values such as:

- `photos/2024/img1.jpg`
- derived from File API relative paths (e.g., `webkitRelativePath`)

The server MUST treat all relpath values as untrusted.

## Normalization rules (MUST)

Given an input relpath:

1. Replace backslashes with forward slashes:
   - `\` -> `/`

2. Reject if:
   - empty
   - contains NUL
   - starts with `/`
   - starts with `~/`
   - contains Windows drive prefix like `C:`
   - contains any path segment `..`

3. Clean using a POSIX clean algorithm:
   - collapse `a//b` to `a/b`
   - collapse `a/./b` to `a/b`

4. Join with destination:
   - `final_abs = join(dest_abs, cleaned_relpath)`

5. Verify containment:
   - `final_abs` MUST be within `dest_abs` after resolving/cleaning.
   - If not, reject the request.

## Test cases (MUST pass)

Reject:
- `../etc/passwd`
- `a/../../b`
- `/absolute/path`
- `C:\Windows\System32`
- `..`
- `a/..`
- `a\..\b`
- `a/../b`

Accept:
- `a/b/c.txt`
- `a//b///c.txt` -> `a/b/c.txt`
- `a/./b/c.txt` -> `a/b/c.txt`
