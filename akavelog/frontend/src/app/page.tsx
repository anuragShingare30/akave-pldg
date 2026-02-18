'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  createInput,
  deleteInput,
  getInputs,
  getIngestBaseUrlForInput,
  getLogsFromIngest,
  getTypeInfo,
  getUploadStatus,
  sendTestLog,
  updateInput,
  type InputItem,
  type InputTypeInfo,
  type LogEntry,
  type UploadStatus as UploadStatusType,
} from '@/lib/api';

export default function DemoPage() {
  const [inputs, setInputs] = useState<InputItem[]>([]);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [uploadStatus, setUploadStatus] = useState<UploadStatusType | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [httpTypeInfo, setHttpTypeInfo] = useState<InputTypeInfo | null>(null);
  const [newTitle, setNewTitle] = useState('my-http-input');
  const [formValues, setFormValues] = useState<Record<string, string>>({});
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [editFormValues, setEditFormValues] = useState<Record<string, string>>({});
  const [updating, setUpdating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadInputs = useCallback(async () => {
    try {
      const { inputs: list } = await getInputs();
      setInputs(list);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load inputs');
    }
  }, []);

  const loadLogs = useCallback(async () => {
    try {
      // GET ingest endpoint directly (raw HTTP) to fetch logs
      const { logs: list } = await getLogsFromIngest('raw');
      setLogs(list);
    } catch {
      // ignore
    }
  }, []);

  const loadStatus = useCallback(async () => {
    try {
      const st = await getUploadStatus();
      setUploadStatus(st);
    } catch {
      setUploadStatus(null);
    }
  }, []);

  useEffect(() => {
    loadInputs();
  }, [loadInputs]);

  useEffect(() => {
    getTypeInfo('http')
      .then((info) => {
        setHttpTypeInfo(info);
        const initial: Record<string, string> = {};
        info.fields.forEach((f) => {
          initial[f.name] = f.example ?? '';
        });
        setFormValues(initial);
      })
      .catch(() => setHttpTypeInfo(null));
  }, []);

  useEffect(() => {
    const t = setInterval(() => {
      loadLogs();
      loadStatus();
    }, 2000);
    return () => clearInterval(t);
  }, [loadLogs, loadStatus]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setError(null);
    try {
      const config: Record<string, unknown> = {};
      Object.entries(formValues).forEach(([k, v]) => {
        const trimmed = typeof v === 'string' ? v.trim() : v;
        if (trimmed !== '') config[k] = trimmed;
      });
      await createInput({
        type: 'http',
        title: newTitle.trim() || undefined,
        config: Object.keys(config).length > 0 ? config : undefined,
      });
      await loadInputs();
      setNewTitle('');
      if (httpTypeInfo) {
        const reset: Record<string, string> = {};
        httpTypeInfo.fields.forEach((f) => {
          reset[f.name] = f.example ?? '';
        });
        setFormValues(reset);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Create failed');
    } finally {
      setCreating(false);
    }
  };

  const handleSendTest = async (input: InputItem) => {
    const path = ingestPath(input);
    const baseUrl = getIngestBaseUrlForInput(input);
    try {
      await sendTestLog(
        path,
        {
          service: 'demo-ui',
          message: `Test log at ${new Date().toISOString()}`,
          level: 'info',
          tags: { source: 'web' },
        },
        { baseUrl: baseUrl ?? undefined }
      );
      await loadLogs();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Send failed');
    }
  };

  const startEdit = (inp: InputItem) => {
    setEditingId(inp.id);
    setEditTitle(inp.title);
    const cfg = (inp.configuration || {}) as Record<string, string>;
    if (httpTypeInfo) {
      const values: Record<string, string> = {};
      httpTypeInfo.fields.forEach((f) => {
        values[f.name] = cfg[f.name] ?? f.example ?? '';
      });
      setEditFormValues(values);
    }
  };

  const cancelEdit = () => {
    setEditingId(null);
  };

  const handleUpdate = async (e: React.FormEvent, id: string) => {
    e.preventDefault();
    setUpdating(true);
    setError(null);
    try {
      const config: Record<string, unknown> = {};
      Object.entries(editFormValues).forEach(([k, v]) => {
        const trimmed = typeof v === 'string' ? v.trim() : v;
        if (trimmed !== '') config[k] = trimmed;
      });
      await updateInput(id, {
        title: editTitle.trim() || undefined,
        config: Object.keys(config).length > 0 ? config : undefined,
      });
      await loadInputs();
      setEditingId(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Update failed');
    } finally {
      setUpdating(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this input? It will stop receiving logs.')) return;
    setDeletingId(id);
    setError(null);
    try {
      await deleteInput(id);
      await loadInputs();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed');
    } finally {
      setDeletingId(null);
    }
  };

  const ingestPath = (input: InputItem) => {
    const cfg = input.configuration as { description?: string; listen?: string };
    if (cfg?.listen) return ''; // dedicated port: path is just /ingest
    return (cfg?.description as string) || 'raw';
  };

  const ingestDisplayPath = (input: InputItem) => {
    const cfg = input.configuration as { listen?: string };
    if (cfg?.listen) return '/ingest';
    return `/ingest/${ingestPath(input) || 'raw'}`;
  };

  return (
    <div className="min-h-screen flex flex-col md:flex-row gap-4 p-4 bg-[var(--bg)]">
      {/* Main content */}
      <div className="flex-1 flex flex-col gap-4 min-w-0">
        <header className="border-b border-[var(--border)] pb-2">
          <h1 className="text-xl font-semibold text-[var(--accent)]">Akavelog Demo</h1>
          <p className="text-sm text-[var(--muted)]">Create HTTP input, send logs, watch uploads to Akave O3</p>
        </header>

        {error && (
          <div className="rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 px-3 py-2 text-sm">
            {error}
          </div>
        )}

        {/* Create HTTP input — form driven by backend type config */}
        <section className="rounded-xl bg-[var(--card)] border border-[var(--border)] p-4">
          <h2 className="text-sm font-medium text-[var(--muted)] mb-3">1. Create HTTP input</h2>
          {!httpTypeInfo ? (
            <p className="text-sm text-[var(--muted)]">Loading input config…</p>
          ) : (
            <form onSubmit={handleCreate} className="flex flex-wrap items-end gap-3">
              <label className="flex flex-col gap-1">
                <span className="text-xs text-[var(--muted)]">Title</span>
                <input
                  type="text"
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  placeholder="my-http-input"
                  className="rounded-lg bg-[var(--bg)] border border-[var(--border)] px-3 py-2 text-sm w-40"
                />
              </label>
              {httpTypeInfo.fields.map((field) => (
                <label key={field.name} className="flex flex-col gap-1">
                  <span className="text-xs text-[var(--muted)]">
                    {field.description}
                    {field.required ? ' *' : ''}
                  </span>
                  <input
                    type={field.type === 'number' ? 'number' : 'text'}
                    value={formValues[field.name] ?? ''}
                    onChange={(e) =>
                      setFormValues((prev) => ({ ...prev, [field.name]: e.target.value }))
                    }
                    placeholder={field.example}
                    className="rounded-lg bg-[var(--bg)] border border-[var(--border)] px-3 py-2 text-sm w-40"
                  />
                </label>
              ))}
              <button
                type="submit"
                disabled={creating}
                className="rounded-lg bg-[var(--accent)] text-[var(--bg)] px-4 py-2 text-sm font-medium disabled:opacity-50"
              >
                {creating ? 'Creating…' : 'Create input'}
              </button>
            </form>
          )}
        </section>

        {/* Inputs list */}
        <section className="rounded-xl bg-[var(--card)] border border-[var(--border)] p-4 flex-1 min-h-0 flex flex-col">
          <h2 className="text-sm font-medium text-[var(--muted)] mb-3">2. Your inputs</h2>
          {inputs.length === 0 ? (
            <p className="text-sm text-[var(--muted)]">Create an input above. Then use “Send test log” to post to /ingest/raw.</p>
          ) : (
            <ul className="space-y-2 overflow-auto">
              {inputs.map((inp) => (
                <li
                  key={inp.id}
                  className="rounded-lg bg-[var(--bg)] border border-[var(--border)] overflow-hidden"
                >
                  {editingId === inp.id && httpTypeInfo ? (
                    <form
                      onSubmit={(e) => handleUpdate(e, inp.id)}
                      className="p-3 space-y-3"
                    >
                      <div className="flex flex-wrap items-end gap-3">
                        <label className="flex flex-col gap-1">
                          <span className="text-xs text-[var(--muted)]">Title</span>
                          <input
                            type="text"
                            value={editTitle}
                            onChange={(e) => setEditTitle(e.target.value)}
                            placeholder="my-http-input"
                            className="rounded-lg bg-[var(--card)] border border-[var(--border)] px-3 py-2 text-sm w-40"
                          />
                        </label>
                        {httpTypeInfo.fields.map((field) => (
                          <label key={field.name} className="flex flex-col gap-1">
                            <span className="text-xs text-[var(--muted)]">{field.description}</span>
                            <input
                              type={field.type === 'number' ? 'number' : 'text'}
                              value={editFormValues[field.name] ?? ''}
                              onChange={(e) =>
                                setEditFormValues((prev) => ({ ...prev, [field.name]: e.target.value }))
                              }
                              placeholder={field.example}
                              className="rounded-lg bg-[var(--card)] border border-[var(--border)] px-3 py-2 text-sm w-40"
                            />
                          </label>
                        ))}
                      </div>
                      <div className="flex gap-2">
                        <button
                          type="submit"
                          disabled={updating}
                          className="rounded-lg bg-[var(--accent)] text-[var(--bg)] px-3 py-1.5 text-sm font-medium disabled:opacity-50"
                        >
                          {updating ? 'Saving…' : 'Save'}
                        </button>
                        <button
                          type="button"
                          onClick={cancelEdit}
                          className="rounded-lg bg-[var(--border)] hover:bg-[var(--muted)] px-3 py-1.5 text-sm"
                        >
                          Cancel
                        </button>
                      </div>
                    </form>
                  ) : (
                    <div className="flex items-center justify-between gap-2 px-3 py-2 text-sm flex-wrap">
                      <span className="font-mono text-[var(--accent)]">{inp.title}</span>
                      <span className="text-[var(--muted)]">{ingestDisplayPath(inp)}</span>
                      <span className="text-xs text-[var(--success)]">{inp.state}</span>
                      <div className="flex gap-1.5">
                        <button
                          type="button"
                          onClick={() => handleSendTest(inp)}
                          className="rounded bg-[var(--border)] hover:bg-[var(--accent)] hover:text-[var(--bg)] px-2 py-1 text-xs"
                        >
                          Send test log
                        </button>
                        <button
                          type="button"
                          onClick={() => startEdit(inp)}
                          className="rounded bg-[var(--border)] hover:bg-[var(--accent)] hover:text-[var(--bg)] px-2 py-1 text-xs"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(inp.id)}
                          disabled={deletingId === inp.id}
                          className="rounded bg-red-500/20 text-red-400 hover:bg-red-500/40 px-2 py-1 text-xs disabled:opacity-50"
                        >
                          {deletingId === inp.id ? 'Deleting…' : 'Delete'}
                        </button>
                      </div>
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* Incoming logs */}
        <section className="rounded-xl bg-[var(--card)] border border-[var(--border)] p-4 flex-1 min-h-[200px] flex flex-col">
          <h2 className="text-sm font-medium text-[var(--muted)] mb-3">3. Incoming logs (last 200)</h2>
          <div className="flex-1 overflow-auto rounded-lg bg-[var(--bg)] border border-[var(--border)] p-2 font-mono text-xs">
            {logs.length === 0 ? (
              <p className="text-[var(--muted)]">Logs fetched via GET /ingest/raw. Send to /ingest/raw then they appear here (polling every 2s).</p>
            ) : (
              <ul className="space-y-2">
                {[...logs].reverse().map((l, i) => (
                  <li key={i} className="border-b border-[var(--border)]/50 pb-2">
                    {l.entry.raw_request ? (
                      <pre className="text-xs whitespace-pre-wrap break-all overflow-x-auto bg-[var(--bg)]/80 p-2 rounded border border-[var(--border)]/50">
                        {l.entry.raw_request.method} {l.entry.raw_request.path}
                        {l.entry.raw_request.query ? `?${l.entry.raw_request.query}` : ''}
                        {l.entry.raw_request.headers && Object.keys(l.entry.raw_request.headers).length > 0 && (
                          <>
                            {'\n'}
                            {Object.entries(l.entry.raw_request.headers).map(([k, v]) => (
                              <span key={k}>{'\n'}{k}: {v}</span>
                            ))}
                          </>
                        )}
                        {l.entry.raw_request.body != null && l.entry.raw_request.body !== '' && (
                          <>{'\n\n'}{l.entry.raw_request.body}</>
                        )}
                      </pre>
                    ) : (
                      <>
                        <span className="text-[var(--muted)]">{new Date(l.received_at).toLocaleTimeString()}</span>
                        {' '}
                        <span className="text-[var(--warn)]">{l.entry.service}</span>
                        {' '}
                        <span className="text-[var(--accent)]">{l.entry.level}</span>
                        {' '}
                        {l.entry.message}
                        {l.entry.tags && Object.keys(l.entry.tags).length > 0 && (
                          <span className="text-[var(--muted)]"> {JSON.stringify(l.entry.tags)}</span>
                        )}
                      </>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>
      </div>

      {/* Side panel: upload status */}
      <aside className="w-full md:w-80 shrink-0 rounded-xl bg-[var(--card)] border border-[var(--border)] p-4 h-fit">
        <h2 className="text-sm font-medium text-[var(--muted)] mb-3">Upload status (O3)</h2>
        {!uploadStatus ? (
          <p className="text-sm text-[var(--muted)]">Loading…</p>
        ) : (
          <div className="space-y-3 text-sm">
            <p>
              Batcher:{' '}
              <span className={uploadStatus.batcher_enabled ? 'text-[var(--success)]' : 'text-[var(--muted)]'}>
                {uploadStatus.batcher_enabled ? 'On' : 'Off'}
              </span>
            </p>
            {uploadStatus.batcher_enabled && (
              <>
                <p className="text-[var(--muted)]">
                  Last upload: {uploadStatus.last_upload_count} logs
                </p>
                {uploadStatus.last_upload_at && (
                  <p className="text-xs text-[var(--muted)]">
                    {new Date(uploadStatus.last_upload_at).toLocaleString()}
                  </p>
                )}
                {uploadStatus.last_upload_key && (
                  <p className="text-xs font-mono text-[var(--accent)] break-all">
                    {uploadStatus.last_upload_key}
                  </p>
                )}
              </>
            )}
          </div>
        )}
      </aside>
    </div>
  );
}
