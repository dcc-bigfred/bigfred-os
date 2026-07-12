export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    readonly detail?: string,
  ) {
    super(detail ?? code);
    this.name = "ApiError";
  }
}

export interface HubSupervisordProgram {
  name: string;
  group?: string;
  command?: string;
  autostart: boolean;
  status: string;
  pid?: number;
}

export interface HubService {
  id: string;
  name: string;
  script: string;
  running: boolean;
}

export interface CurrentUser {
  username: string;
}

export interface LogEntry {
  id: string;
  root: string;
  service: string;
  name: string;
  size: number;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  let body: Record<string, unknown> = {};
  if (text) {
    try {
      body = JSON.parse(text) as Record<string, unknown>;
    } catch {
      // plain-text error bodies (e.g. chi 404 page)
    }
  }

  if (!res.ok) {
    const code = typeof body.error === "string" ? body.error : `http_${res.status}`;
    const detail =
      typeof body.message === "string"
        ? body.message
        : typeof body.error !== "string" && text
          ? text.trim().slice(0, 200)
          : undefined;
    throw new ApiError(res.status, code, detail);
  }

  if (!text) {
    return undefined as T;
  }
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new ApiError(res.status, "bad_response", "Invalid JSON from server");
  }
}

export function fetchMe(): Promise<CurrentUser> {
  return apiFetch<CurrentUser>("/api/v1/auth/me");
}

export function login(username: string, password: string): Promise<CurrentUser> {
  return apiFetch<CurrentUser>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function logout(): Promise<void> {
  return apiFetch<void>("/api/v1/auth/logout", { method: "POST" });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiFetch<void>("/api/v1/auth/password", {
    method: "POST",
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
}

export function fetchLogs(): Promise<LogEntry[]> {
  return apiFetch<LogEntry[]>("/api/v1/logs");
}

export type LogWSMessage =
  | { type: "history"; lines: string[] }
  | { type: "line"; text: string }
  | { type: "error"; error: string };

export function logStreamURL(id: string): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/api/v1/logs/stream?id=${encodeURIComponent(id)}`;
}

export function terminalStreamURL(): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/api/v1/terminal`;
}

export function fetchServices(): Promise<HubService[]> {
  return apiFetch<HubService[]>("/api/v1/services");
}

export type ServiceAction = "start" | "stop" | "restart";

export function serviceAction(id: string, action: ServiceAction): Promise<void> {
  return apiFetch<void>(`/api/v1/services/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
  });
}

export function fetchSupervisordPrograms(): Promise<HubSupervisordProgram[]> {
  return apiFetch<HubSupervisordProgram[]>("/api/v1/supervisord/programs");
}

export type SupervisordAction = "start" | "stop" | "restart";

export function supervisordProgramAction(name: string, action: SupervisordAction): Promise<void> {
  return apiFetch<void>(
    `/api/v1/supervisord/programs/${encodeURIComponent(name)}/${action}`,
    { method: "POST" },
  );
}

export interface RedisKeySummary {
  key: string;
  ttl: number;
}

export interface RedisKeyDetail {
  key: string;
  type: string;
  ttl: number;
  value: unknown;
}

export function fetchRedisKeys(pattern = "*"): Promise<RedisKeySummary[]> {
  const q = new URLSearchParams({ pattern });
  return apiFetch<RedisKeySummary[]>(`/api/v1/redis/keys?${q}`);
}

export function fetchRedisKey(key: string): Promise<RedisKeyDetail> {
  const q = new URLSearchParams({ key });
  return apiFetch<RedisKeyDetail>(`/api/v1/redis/key?${q}`);
}

export function deleteRedisKey(key: string): Promise<void> {
  const q = new URLSearchParams({ key });
  return apiFetch<void>(`/api/v1/redis/key?${q}`, { method: "DELETE" });
}

export type RedisKeyWSMessage =
  | { type: "snapshot"; detail: RedisKeyDetail }
  | { type: "update"; detail: RedisKeyDetail }
  | { type: "deleted" }
  | { type: "error"; error: string };

export function redisKeyStreamURL(key: string): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/api/v1/redis/stream?key=${encodeURIComponent(key)}`;
}

export interface EtcFile {
  path: string;
  name: string;
  size: number;
  modified: string;
}

export interface EtcFileContent {
  path: string;
  content: string;
  size: number;
  modified: string;
}

export function fetchEtcFiles(): Promise<EtcFile[]> {
  return apiFetch<EtcFile[]>("/api/v1/etc/files");
}

export function fetchEtcFile(path: string): Promise<EtcFileContent> {
  const q = new URLSearchParams({ path });
  return apiFetch<EtcFileContent>(`/api/v1/etc/file?${q}`);
}

export function saveEtcFile(path: string, content: string): Promise<EtcFileContent> {
  const q = new URLSearchParams({ path });
  return apiFetch<EtcFileContent>(`/api/v1/etc/file?${q}`, {
    method: "PUT",
    body: JSON.stringify({ content }),
  });
}
