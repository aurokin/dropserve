
# User Flows

## Flow A — One-time portal, upload files/folders, closes on timer

1. User SSHes into the server (or uses a terminal on the server).
2. User `cd`s into the destination directory.
3. User runs:

   - `dropserve` (default 15 minutes, one-time)

4. CLI prints a URL like:

   - `http://192.168.1.23/p/p_ABC...`

5. User opens the URL on a LAN desktop browser.
6. Browser claims the portal and shows an upload page.
7. User drags files/folders into the page or clicks “Select folder”.
8. UI runs a preflight to detect collisions (optional but recommended).
9. User chooses:
   - overwrite (with warning), or
   - auto-rename conflicts
10. UI uploads queued items sequentially, showing:
    - overall progress
    - transfer speed
11. Server verifies each file and commits atomically to destination.
12. When the queue finishes, the portal remains open until the open window expires.
13. Temporary files are deleted after the portal closes.

## Flow B — Portal expires without being used

1. User runs `dropserve` (15 min).
2. Nobody visits the portal page.
3. After 15 minutes, portal expires.
4. Server cleans portal temp directory and removes portal record.

## Flow C — Expiry hits mid-transfer

1. Portal open window ends while a transfer is in progress.
2. Portal does NOT close mid-transfer.
3. After active uploads complete, the portal closes when the open window expires.

## Flow D — Reusable portal

1. User runs `dropserve --reusable --minutes 120`.
2. Portal can be used multiple times within the duration without claiming (details in `04_Portal_Spec.md`).
3. Portal closes when duration ends AND no active uploads remain, or when closed explicitly.
