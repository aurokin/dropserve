# Overview

## Summary

DropServe is a self-hosted LAN tool that opens a short-lived upload portal to a specific server directory. The CLI runs on the server in the destination directory; browsers never choose absolute paths.

## Primary goals

- "Just works" LAN uploads for desktop browsers.
- No partial files in the final destination.
- Aggressive cleanup of temp artifacts.
- Short-lived, one-time portals by default.
- Sequential upload queue UX with progress and speed.

## Non-goals (v1)

- Resumable uploads.
- Mobile browser support.
- Preserving permissions, ownership, timestamps, extended attributes.
- Sync/backup semantics.

## Key constraints

- OS: Ubuntu LTS.
- Reverse proxy: Caddy only (ports 80/443).
- App binds to all interfaces on a high port (default `0.0.0.0:8080`).
- Typical uploads < 100MB; outliers up to 50GB.
- LAN-only posture; portals are high-entropy capability links.

## User flows (short form)

- **Open + upload**: run `dropserve` in the destination directory, open the LAN link, upload files/folders, portal closes after duration when uploads finish.
- **Unused portal**: portal expires after the open window and is cleaned.
- **Expiry mid-upload**: uploads continue; portal closes after active uploads drain.
- **Reusable portal**: `--reusable` allows multiple uploads during the open window without claiming.

## Success criteria

- Folder uploads with hundreds of files succeed reliably.
- Canceling or crashing leaves no partial files in the destination.
- Overwrite warnings are clear; autorename only when needed.
