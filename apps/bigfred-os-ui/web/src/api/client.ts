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

  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const code = typeof body.error === "string" ? body.error : "unknown";
    const detail = typeof body.message === "string" ? body.message : undefined;
    throw new ApiError(res.status, code, detail);
  }
  return body as T;
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
