import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";

type StatusTone = "info" | "ok" | "error" | "warn";

type StatusState = {
  message: string;
  tone: StatusTone;
};

type QueueItem = {
  id: string;
  file: File;
  relpath: string;
  status: string;
  progress: number;
};

type QueueCandidate = {
  file: File;
  relpath: string;
};

type ClaimPolicy = {
  overwrite: boolean;
  autorename: boolean;
};

type ClaimResponse = {
  portal_id: string;
  client_token: string;
  expires_at: string;
  policy: ClaimPolicy;
  reusable?: boolean;
};

type PreflightConflict = {
  relpath: string;
  reason: string;
};

type DropServeFileSystemEntry = {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
  fullPath: string;
  file: (success: (file: File) => void, error?: (err: DOMException) => void) => void;
  createReader: () => DropServeDirectoryReader;
};

type DropServeDirectoryReader = {
  readEntries: (
    success: (entries: DropServeFileSystemEntry[]) => void,
    error?: (err: DOMException) => void
  ) => void;
};

type DataTransferItemWithEntry = DataTransferItem & {
  webkitGetAsEntry?: () => DropServeFileSystemEntry | null;
};

const fileInputProps = {
  multiple: true
};

const folderInputProps = {
  multiple: true,
  webkitdirectory: "true",
  directory: "true"
} as React.InputHTMLAttributes<HTMLInputElement>;

const defaultStatus: StatusState = {
  message: "Preparing portal...",
  tone: "info"
};

function App() {
  const portalId = useMemo(() => getPortalId(window.location.pathname), []);
  const isPortalPage = portalId.length > 0;

  return (
    <div className="page">
      <main className="content">
        {isPortalPage ? <PortalPage portalId={portalId} /> : <LandingPage />}
      </main>
    </div>
  );
}

function LandingPage() {
  return (
    <section className="card hero">
      <div className="hero-copy">
        <p className="eyebrow">Quick start</p>
        <h1>Open a portal from your server, then share the link on your LAN.</h1>
        <p className="lead">
          DropServe keeps uploads sequential and safe. Files land only after the
          transfer completes and verifies.
        </p>
        <div className="steps">
          <div className="step">
            <span>1</span>
            <div>
              Run <code>dropserve open</code> in the destination folder.
            </div>
          </div>
          <div className="step">
            <span>2</span>
            <div>Copy the LAN URL from the CLI output.</div>
          </div>
          <div className="step">
            <span>3</span>
            <div>Open the link on a LAN desktop to upload.</div>
          </div>
        </div>
        <div className="callout">
          <div className="callout-title">Tip</div>
          <div className="callout-body">
            Keep the terminal open while uploads are in progress. Close it after
            the portal finishes.
          </div>
        </div>
      </div>
      <div className="hero-panel">
        <div className="hero-panel-header">Example</div>
        <pre>
          <code>$ dropserve open --minutes 20 --policy autorename</code>
        </pre>
        <div className="hero-panel-footer">
          Your CLI prints a short URL like:
          <div className="chip">http://192.168.1.42/p/p_abc123</div>
        </div>
      </div>
    </section>
  );
}

function PortalPage({ portalId }: { portalId: string }) {
  const [status, setStatus] = useState<StatusState>(defaultStatus);
  const [claimed, setClaimed] = useState(false);
  const [running, setRunning] = useState(false);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [conflicts, setConflicts] = useState<PreflightConflict[]>([]);
  const [defaultPolicy, setDefaultPolicy] = useState<"overwrite" | "autorename">(
    "overwrite"
  );
  const [totalBytes, setTotalBytes] = useState(0);
  const [uploadedBytes, setUploadedBytes] = useState(0);
  const [speedBps, setSpeedBps] = useState(0);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [portalReusable, setPortalReusable] = useState(true);

  const clientTokenRef = useRef("");
  const claimAttemptedRef = useRef(false);
  const queueRef = useRef<QueueItem[]>([]);
  const uploadedBytesRef = useRef(0);
  const completedBytesRef = useRef(0);
  const speedTimerRef = useRef<number | null>(null);
  const lastSpeedBytesRef = useRef(0);
  const lastSpeedTimeRef = useRef(0);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const folderInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    queueRef.current = queue;
  }, [queue]);

  const updateUploadedBytes = useCallback((value: number) => {
    uploadedBytesRef.current = value;
    setUploadedBytes(value);
  }, []);

  const updateStatus = useCallback((message: string, tone: StatusTone = "info") => {
    setStatus({ message, tone });
  }, []);

  const updateQueueItem = useCallback(
    (id: string, updates: Partial<Pick<QueueItem, "status" | "progress">>) => {
      setQueue((items) =>
        items.map((item) =>
          item.id === id ? { ...item, ...updates } : item
        )
      );
    },
    []
  );

  const startSpeedTimer = useCallback(() => {
    if (speedTimerRef.current !== null) {
      return;
    }
    lastSpeedBytesRef.current = uploadedBytesRef.current;
    lastSpeedTimeRef.current = performance.now();
    speedTimerRef.current = window.setInterval(() => {
      const now = performance.now();
      const deltaBytes = uploadedBytesRef.current - lastSpeedBytesRef.current;
      const deltaSeconds = (now - lastSpeedTimeRef.current) / 1000;
      if (deltaSeconds > 0) {
        setSpeedBps(Math.max(0, deltaBytes / deltaSeconds));
      }
      lastSpeedBytesRef.current = uploadedBytesRef.current;
      lastSpeedTimeRef.current = now;
    }, 600);
  }, []);

  const stopSpeedTimer = useCallback(() => {
    if (speedTimerRef.current === null) {
      return;
    }
    window.clearInterval(speedTimerRef.current);
    speedTimerRef.current = null;
    setSpeedBps(0);
  }, []);

  const runPreflight = useCallback(
    async (items: QueueItem[] = queueRef.current, showError = false) => {
      if (!claimed || running || items.length === 0) {
        setConflicts([]);
        return true;
      }
      const payload = {
        items: items.map((item) => ({
          relpath: item.relpath,
          size: item.file.size
        }))
      };
      try {
        const response = await fetch(`/api/portals/${portalId}/preflight`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Client-Token": clientTokenRef.current
          },
          body: JSON.stringify(payload)
        });
        if (!response.ok) {
          if (showError) {
            const message = await readError(response);
            updateStatus(`Preflight failed: ${message}`, "error");
          }
          return false;
        }
        const data = await response.json();
        const incoming = Array.isArray(data.conflicts) ? data.conflicts : [];
        setConflicts(incoming);
        return true;
      } catch {
        if (showError) {
          updateStatus("Preflight failed.", "error");
        }
        return false;
      }
    },
    [claimed, portalId, running, updateStatus]
  );

  const addQueueItems = useCallback(
    (items: QueueCandidate[]) => {
      if (!claimed) {
        return;
      }
      const normalized = items.filter((item) => item.file);
      if (normalized.length === 0) {
        return;
      }
      setQueue((existing) => {
        const next = [...existing];
        let addedBytes = 0;
        for (const item of normalized) {
          const relpath = item.relpath || item.file.webkitRelativePath || item.file.name;
          next.push({
            id: makeLocalID(),
            file: item.file,
            relpath,
            status: "queued",
            progress: 0
          });
          addedBytes += item.file.size;
        }
        setTotalBytes((value) => value + addedBytes);
        runPreflight(next.filter((item) => item.status === "queued"), false);
        return next;
      });
    },
    [claimed, runPreflight]
  );

  const claimPortal = useCallback(async () => {
    if (!portalId) {
      updateStatus("Invalid portal URL.", "error");
      return;
    }
    updateStatus("Claiming portal...", "info");
    try {
      const response = await fetch(`/api/portals/${portalId}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: "{}"
      });
      if (!response.ok) {
        const message = await readError(response);
        updateStatus(message, "error");
        return;
      }
      const data: ClaimResponse = await response.json();
      clientTokenRef.current = data.client_token;
      setExpiresAt(data.expires_at || null);
      const policy = data.policy && data.policy.autorename ? "autorename" : "overwrite";
      setDefaultPolicy(policy);
      setPortalReusable(data.reusable !== false);
      setClaimed(true);
      updateStatus("Portal ready. Drop or click to add files.", "ok");
      runPreflight(queueRef.current, false);
    } catch {
      updateStatus("Failed to claim portal.", "error");
    }
  }, [portalId, runPreflight, updateStatus]);

  const initUpload = useCallback(
    async (item: QueueItem) => {
      const payload = {
        upload_id: makeUploadID(),
        relpath: item.relpath,
        size: item.file.size,
        client_sha256: null,
        policy: defaultPolicy
      };
      const response = await fetch(`/api/portals/${portalId}/uploads`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Client-Token": clientTokenRef.current
        },
        body: JSON.stringify(payload)
      });
      if (!response.ok) {
        const message = await readError(response);
        throw new Error(message);
      }
      return response.json();
    },
    [defaultPolicy, portalId]
  );

  const putUpload = useCallback(
    (item: QueueItem, putUrl: string) =>
      new Promise<void>((resolve, reject) => {
        const request = new XMLHttpRequest();
        request.open("PUT", putUrl);
        request.setRequestHeader("X-Client-Token", clientTokenRef.current);
        request.upload.onprogress = (event) => {
          if (!event.lengthComputable) {
            return;
          }
          const current = completedBytesRef.current + event.loaded;
          updateUploadedBytes(current);
          const percent = item.file.size > 0 ? Math.round((event.loaded / item.file.size) * 100) : 100;
          updateQueueItem(item.id, { status: "uploading", progress: percent });
        };
        request.onload = () => {
          if (request.status >= 200 && request.status < 300) {
            resolve();
            return;
          }
          let message = request.statusText || "upload failed";
          try {
            const data = JSON.parse(request.responseText);
            if (data && data.error) {
              message = data.error;
            }
          } catch {
          }
          reject(new Error(message));
        };
        request.onerror = () => reject(new Error("network error"));
        request.send(item.file);
      }),
    [updateQueueItem, updateUploadedBytes]
  );

  const closePortal = useCallback(async () => {
    try {
      const response = await fetch(`/api/portals/${portalId}/close`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Client-Token": clientTokenRef.current
        },
        body: "{}"
      });
      if (!response.ok) {
        const message = await readError(response);
        updateStatus(`Uploads complete. Portal close failed: ${message}`, "warn");
        return;
      }
      updateStatus("All uploads complete. Portal closed.", "ok");
    } catch {
      updateStatus("Uploads complete. Portal close failed.", "warn");
    }
  }, [portalId, updateStatus]);

  const runQueue = useCallback(async () => {
    if (running || !claimed || queueRef.current.length === 0) {
      return;
    }
    const pendingItems = queueRef.current.filter((item) => item.status === "queued");
    if (pendingItems.length === 0) {
      return;
    }
    const preflightOk = await runPreflight(pendingItems, true);
    if (!preflightOk) {
      return;
    }
    setRunning(true);
    const completedBytes = queueRef.current.reduce((sum, item) => {
      if (item.status === "done") {
        return sum + item.file.size;
      }
      return sum;
    }, 0);
    completedBytesRef.current = completedBytes;
    updateUploadedBytes(completedBytes);
    setSpeedBps(0);
    startSpeedTimer();

    for (const item of pendingItems) {
      updateQueueItem(item.id, { status: "initializing", progress: 0 });
      let initResponse;
      try {
        initResponse = await initUpload(item);
      } catch (error) {
        const message = error instanceof Error ? error.message : "upload failed";
        updateQueueItem(item.id, { status: "failed" });
        updateStatus(`Upload failed: ${message}`, "error");
        setRunning(false);
        stopSpeedTimer();
        return;
      }

      updateQueueItem(item.id, { status: "uploading", progress: 0 });
      try {
        await putUpload(item, initResponse.put_url);
      } catch (error) {
        const message = error instanceof Error ? error.message : "upload failed";
        updateQueueItem(item.id, { status: "failed" });
        updateStatus(`Upload failed: ${message}`, "error");
        setRunning(false);
        stopSpeedTimer();
        return;
      }

      updateQueueItem(item.id, { status: "done", progress: 100 });
      completedBytesRef.current += item.file.size;
      updateUploadedBytes(completedBytesRef.current);
    }

    setRunning(false);
    stopSpeedTimer();
    if (!portalReusable) {
      await closePortal();
    } else {
      updateStatus("All uploads complete. Portal remains open.", "ok");
    }
  }, [
    claimed,
    closePortal,
    initUpload,
    portalReusable,
    putUpload,
    runPreflight,
    running,
    startSpeedTimer,
    stopSpeedTimer,
    updateQueueItem,
    updateStatus,
    updateUploadedBytes
  ]);

  useEffect(() => {
    if (!portalId || claimed || claimAttemptedRef.current) {
      return;
    }
    claimAttemptedRef.current = true;
    claimPortal();
  }, [claimed, claimPortal, portalId]);

  useEffect(() => () => stopSpeedTimer(), [stopSpeedTimer]);

  const handleFileInput = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const items = normalizeFileList(event.target.files);
      addQueueItems(items);
      event.target.value = "";
    },
    [addQueueItems]
  );

  const handleDrop = useCallback(
    async (event: React.DragEvent<HTMLDivElement>) => {
      if (!claimed) {
        return;
      }
      event.preventDefault();
      event.currentTarget.classList.remove("dragging");
      const items = await collectDropItems(event.dataTransfer);
      addQueueItems(items);
    },
    [addQueueItems, claimed]
  );

  const handleDragOver = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      if (!claimed) {
        return;
      }
      event.preventDefault();
      event.currentTarget.classList.add("dragging");
    },
    [claimed]
  );

  const handleDragLeave = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.currentTarget.classList.remove("dragging");
  }, []);

  const queuedCount = queue.filter((item) => item.status === "queued").length;
  const conflictCount = conflicts.length;
  const conflictVerb = defaultPolicy === "autorename" ? "auto-renamed" : "overwritten";
  const expiryLabel = expiresAt ? formatTimestamp(expiresAt) : "";
  const overallProgress = totalBytes > 0 ? Math.min(100, Math.round((uploadedBytes / totalBytes) * 100)) : 0;

  const handleChooseFiles = useCallback(() => {
    if (!claimed) {
      return;
    }
    fileInputRef.current?.click();
  }, [claimed]);

  const handleChooseFolders = useCallback(() => {
    if (!claimed) {
      return;
    }
    folderInputRef.current?.click();
  }, [claimed]);

  useEffect(() => {
    if (!claimed || running || queuedCount === 0) {
      return;
    }
    runQueue();
  }, [claimed, queuedCount, runQueue, running]);

  return (
    <div className="portal-layout">
      <section className="portal-splash">
        <div
          className={`splash-drop ${!claimed ? "disabled" : ""}`}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
        >
          <div className="splash-top">
            <div className="meta">
              Portal {portalId}
              {expiryLabel && <span className="meta-divider">•</span>}
              {expiryLabel && `Expires ${expiryLabel}`}
            </div>
            <div className={`status status-${status.tone}`}>{status.message}</div>
          </div>
          <div className="splash-center">
            <div className="splash-title">Drag files or folders here</div>
            <div className="splash-subtitle">Use the buttons below to choose files or folders.</div>
            <div className="splash-note">Uploads begin automatically. No upload button needed.</div>
          </div>
          <div className="picker-actions">
            <button
              type="button"
              className="picker-button"
              onClick={handleChooseFiles}
              disabled={!claimed}
            >
              <span className="picker-icon" aria-hidden="true">
                <svg viewBox="0 0 20 20" focusable="false" aria-hidden="true">
                  <path d="M6 2.5h5l4 4v11H6z" fill="none" stroke="currentColor" strokeWidth="1.5" />
                  <path d="M11 2.5v4h4" fill="none" stroke="currentColor" strokeWidth="1.5" />
                </svg>
              </span>
              Choose files
            </button>
            <button
              type="button"
              className="picker-button"
              onClick={handleChooseFolders}
              disabled={!claimed}
            >
              <span className="picker-icon" aria-hidden="true">
                <svg viewBox="0 0 20 20" focusable="false" aria-hidden="true">
                  <path d="M2.5 6.5h6l1.6 2h7.4v8.5H2.5z" fill="none" stroke="currentColor" strokeWidth="1.5" />
                  <path d="M2.5 6.5v-2h5l1.6 2" fill="none" stroke="currentColor" strokeWidth="1.5" />
                </svg>
              </span>
              Choose folders
            </button>
          </div>
          <input
            {...fileInputProps}
            ref={fileInputRef}
            type="file"
            className="splash-input"
            disabled={!claimed}
            onChange={handleFileInput}
          />
          <input
            {...folderInputProps}
            ref={folderInputRef}
            type="file"
            className="splash-input"
            disabled={!claimed}
            onChange={handleFileInput}
          />
        </div>

        <div className="progress-panel">
          <div className="progress-header">
            <div className="progress-title">Overall progress</div>
            <div className="progress-value">{overallProgress}%</div>
          </div>
          <div className="progress-track">
            <div className="progress-bar" style={{ width: `${overallProgress}%` }} />
          </div>
          <div className="progress-meta">
            <div>{formatBytes(uploadedBytes)} / {formatBytes(totalBytes)}</div>
            <div>{queuedCount} queued · {queue.length} total</div>
            <div>{formatBytes(speedBps)}/s</div>
          </div>
        </div>

        <div className={`conflict-panel ${conflictCount === 0 ? "hidden" : ""}`}>
          <div className="conflict-title">Filename conflicts detected</div>
          <div className="conflict-message">
            {conflictCount} {conflictCount === 1 ? "file" : "files"} already exist and will be {conflictVerb}.
          </div>
          <label className="toggle">
            <input
              type="checkbox"
              checked={defaultPolicy === "autorename"}
              disabled={!claimed || running || conflictCount === 0}
              onChange={(event) =>
                setDefaultPolicy(event.target.checked ? "autorename" : "overwrite")
              }
            />
            Auto-rename conflicts instead
          </label>
        </div>

        <div className="queue-card">
          <div className="queue-header">
            <h2>Queue</h2>
            <span className="queue-meta">{queue.length} items</span>
          </div>
          {queue.length > 0 && (
            <div className="queue-list">
              {queue.map((item) => {
                const progress = Math.min(100, Math.max(0, item.progress));
                const statusLabel =
                  item.status === "queued"
                    ? "Queued"
                    : item.status === "initializing"
                      ? "Starting"
                      : item.status === "uploading"
                        ? "Uploading"
                        : item.status === "done"
                          ? "Done"
                          : item.status === "failed"
                            ? "Failed"
                            : item.status;
                return (
                  <div key={item.id} className={`queue-row status-${item.status}`}>
                    <div className="queue-main">
                      <div className="queue-name" title={item.relpath}>
                        {item.relpath}
                      </div>
                      <div className="queue-progress">
                        <div className="queue-bar" style={{ width: `${progress}%` }} />
                      </div>
                    </div>
                    <div className="queue-status">
                      <span>{statusLabel}</span>
                      <span className="queue-percent">{progress}%</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function getPortalId(pathname: string) {
  const match = pathname.match(/^\/p\/([^/]+)/);
  return match ? match[1] : "";
}

function makeLocalID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `q_${Math.random().toString(16).slice(2)}${Date.now().toString(16)}`;
}

function makeUploadID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `u_${Math.random().toString(16).slice(2)}${Date.now().toString(16)}`;
}

function normalizeFileList(files: FileList | null): QueueCandidate[] {
  if (!files) {
    return [];
  }
  return Array.from(files).map((file) => ({
    file,
    relpath: file.webkitRelativePath || file.name
  }));
}

async function collectDropItems(dataTransfer: DataTransfer | null): Promise<QueueCandidate[]> {
  if (!dataTransfer) {
    return [];
  }
  const items = Array.from(dataTransfer.items || []);
  if (items.length === 0) {
    return normalizeFileList(dataTransfer.files);
  }
  const files: QueueCandidate[] = [];
  for (const item of items) {
    if (item.kind !== "file") {
      continue;
    }
    const entry = (item as DataTransferItemWithEntry).webkitGetAsEntry?.() as
      | DropServeFileSystemEntry
      | null;
    if (entry) {
      const entryFiles = await readEntryFiles(entry);
      files.push(...entryFiles);
      continue;
    }
    const file = item.getAsFile?.();
    if (file) {
      files.push({ file, relpath: file.webkitRelativePath || file.name });
    }
  }
  return files;
}

async function readEntryFiles(entry: DropServeFileSystemEntry): Promise<QueueCandidate[]> {
  if (entry.isFile) {
    return new Promise((resolve) => {
      entry.file(
        (file) => {
          const relpath = stripLeadingSlash(entry.fullPath) || file.webkitRelativePath || file.name;
          resolve([{ file, relpath }]);
        },
        () => resolve([])
      );
    });
  }
  if (entry.isDirectory) {
    const reader = entry.createReader();
    const entries = await readDirectoryEntries(reader);
    const files: QueueCandidate[] = [];
    for (const child of entries) {
      const childFiles = await readEntryFiles(child);
      files.push(...childFiles);
    }
    return files;
  }
  return [];
}

function readDirectoryEntries(reader: DropServeDirectoryReader): Promise<DropServeFileSystemEntry[]> {
  return new Promise((resolve) => {
    const entries: DropServeFileSystemEntry[] = [];
    const readBatch = () => {
      reader.readEntries(
        (batch) => {
          if (batch.length === 0) {
            resolve(entries);
            return;
          }
          entries.push(...batch);
          readBatch();
        },
        () => resolve(entries)
      );
    };
    readBatch();
  });
}

function stripLeadingSlash(value: string) {
  return value.replace(/^\/+/, "");
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const digits = value >= 10 || unitIndex === 0 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

function formatTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) {
    return value;
  }
  return date.toLocaleString();
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    if (data && data.error) {
      return data.error as string;
    }
  } catch {
  }
  return response.statusText || "request failed";
}

export default App;
