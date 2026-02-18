const API = process.env.NEXT_PUBLIC_API_URL || '/api';

// Standard API response shape (success)
export type APIResponse<T = unknown> = {
  data: T;
  status: number;
  message?: string;
  path: string;
};

// Standard API error shape
export type APIError = {
  message: string;
  error: string;
  path: string;
  status: number;
};

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const r = await fetch(url, init);
  const body = await r.json().catch(() => ({}));
  if (!r.ok) {
    const err = body as Partial<APIError>;
    const msg = err?.message || err?.error || r.statusText || 'Request failed';
    throw new Error(msg);
  }
  const res = body as APIResponse<T>;
  return res.data !== undefined ? res.data : (body as T);
}

export type ConfigField = {
  name: string;
  type: string;
  required: boolean;
  description: string;
  example?: string;
};

export type InputTypeInfo = {
  type: string;
  description: string;
  fields: ConfigField[];
};

export type InputItem = {
  id: string;
  type: string;
  title: string;
  configuration: Record<string, unknown>;
  created_at: string;
  state: string;
};

export type RawRequestData = {
  method: string;
  path: string;
  query?: string;
  headers?: Record<string, string>;
  body?: string;
};

export type LogEntry = {
  entry: {
    timestamp: string;
    service: string;
    level: string;
    message: string;
    tags?: Record<string, string>;
    raw_request?: RawRequestData;
  };
  received_at: string;
};

export type UploadStatus = {
  batcher_enabled: boolean;
  last_upload_at: string;
  last_upload_key: string;
  last_upload_count: number;
  pending_count: number;
};

export async function getInputTypes(): Promise<{ types: string[] }> {
  return request<{ types: string[] }>(`${API}/inputs/types`);
}

export async function getTypeInfo(typeName: string): Promise<InputTypeInfo> {
  return request<InputTypeInfo>(`${API}/inputs/types/${encodeURIComponent(typeName)}`);
}

export async function getInputs(): Promise<{ inputs: InputItem[] }> {
  return request<{ inputs: InputItem[] }>(`${API}/inputs`);
}

export async function createInput(body: {
  type: string;
  title?: string;
  description?: string;
  listen?: string;
  config?: Record<string, unknown>;
}): Promise<InputItem> {
  return request<InputItem>(`${API}/inputs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export async function updateInput(
  id: string,
  body: {
    title?: string;
    description?: string;
    listen?: string;
    config?: Record<string, unknown>;
  }
): Promise<InputItem> {
  return request<InputItem>(`${API}/inputs/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export async function deleteInput(id: string): Promise<void> {
  await request<null>(`${API}/inputs/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function getRecentLogs(): Promise<{ logs: LogEntry[] }> {
  return request<{ logs: LogEntry[] }>(`${API}/logs/recent`);
}

/** GET the ingest endpoint directly (raw HTTP); returns recent logs with full response details. */
export async function getLogsFromIngest(ingestPath: string): Promise<{ logs: LogEntry[] }> {
  const path = ingestPath.replace(/^\/+|\/+$/g, '') || 'raw';
  return request<{ logs: LogEntry[] }>(`${API}/ingest/${path}`);
}

export async function getUploadStatus(): Promise<UploadStatus> {
  return request<UploadStatus>(`${API}/logs/status`);
}

export function getIngestUrl(path: string): string {
  const base = process.env.NEXT_PUBLIC_API_URL || '';
  return `${base || ''}/api/ingest/${path}`.replace(/\/+/g, '/');
}

/**
 * When input has a custom listen port (e.g. ":9001"), returns the base URL to use for ingest.
 * Otherwise returns null (use main API). Only valid in browser (uses window.location).
 * Always returns a full absolute URL (http(s)://host:port) so fetch never gets a broken URL.
 */
export function getIngestBaseUrlForInput(input: InputItem): string | null {
  const listen = input?.configuration?.listen as string | undefined;
  if (!listen || typeof listen !== 'string') return null;
  const port = listen.trim().split(':').pop();
  if (!port || !/^\d+$/.test(port)) return null;
  if (typeof window === 'undefined') return null;
  const protocol = window.location?.protocol || 'http:';
  const host = window.location?.hostname || 'localhost';
  const base = `${protocol}//${host}:${port}`;
  return base.startsWith('http') ? base : null;
}

export async function sendTestLog(
  ingestPath: string,
  payload: object,
  options?: { baseUrl?: string | null }
): Promise<void> {
  const path = ingestPath.replace(/^\/+|\/+$/g, '');
  const base = options?.baseUrl ?? API;
  const pathPart = path ? `/${path}` : '';
  const url =
    base && base.startsWith('http')
      ? `${base.replace(/\/+$/, '')}/ingest${pathPart}`
      : `${API}/ingest${pathPart || '/raw'}`;
  const r = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const body = await r.json().catch(() => ({}));
  if (!r.ok) {
    const err = body as Partial<APIError>;
    throw new Error(err?.message || err?.error || r.statusText || 'Send failed');
  }
}
