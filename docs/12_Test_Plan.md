
# Test Plan

This plan focuses on file safety, cleanup, and portal timing rules.

## Manual test matrix (must pass for v1)

### A) Upload success
- Create portal via CLI
- Upload a folder with 100–300 files
- Verify all files present under destination
- Verify server reports SHA-256 and size match

### B) Cancel mid-upload (browser closes tab)
- Start uploading a large file (>= 500MB if possible)
- Close the tab mid-transfer
- Expected:
  - no final file exists in destination
  - temp `.part` is deleted immediately OR within sweep interval
  - portal remains consistent (active_upload count decremented)

### C) Server kill during upload
- Start uploading a large file
- Kill server process (SIGKILL)
- Restart server
- Expected:
  - no final partial file in destination
  - sweeper removes `.part` remnants on startup

### D) Portal expires mid-transfer
- Create portal with short duration (e.g., 1 minute)
- Start uploading a large file
- Wait for timer to elapse
- Expected:
  - upload continues and completes
  - portal closes after upload finishes (one-time portal)

### E) Overwrite warning and autorename
- Prepare destination with existing file(s)
- Upload same name
- Expected:
  - overwrite warns user
  - autorename produces `name_YYYY-MM-DD_HHMMSS.ext` only when needed
  - multiple conflicts in same second add `_2`, `_3`, etc.

## Automated tests (recommended)

- Unit tests for:
  - portal state machine transitions (see `04_Portal_Spec.md`)
  - path safety normalization/rejection (see `07_Path_Safety.md`)
  - autorename function behavior
- Integration tests (can be in-process HTTP):
  - upload init -> PUT -> commit
  - cancel request -> cleanup
  - crash simulation -> startup sweep
