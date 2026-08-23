export interface StatusResponse {
  running: boolean;
  synchronized: boolean;
  stratum: number;
  sourceCount: number;
  reachableSourceCount: number;
  clientCount: number;
  serverTimeUtc: string;
}

export interface TrackingResponse {
  refId: string;
  refName: string;
  stratum: number;
  systemOffsetSecs: number;
  lastOffsetSecs: number;
  rmsOffsetSecs: number;
  frequencyPpm: number;
  rootDelaySecs: number;
  rootDispersionSecs: number;
  updateIntervalSecs: number;
  leapStatus: string;
  synchronized: boolean;
  refTimeUtc?: string;
  lastSyncAgoSecs?: number;
  serverTimeLocal: string;
  serverTimeUtc: string;
  timezone: string;
}

export interface SourceView {
  address: string;
  stratum: number;
  reach: number;
  lastRxSecs: number;
  offsetSecs: number;
  jitterSecs: number;
  status: string;
}

export interface ClientView {
  address: string;
  ntpRequests: number;
  lastRequestAgoSecs: number | null;
  status: string;
}

export interface DiagnosticCheck {
  name: string;
  status: "ok" | "warn" | "fail";
  message: string;
}

export interface SessionResponse {
  authenticated: boolean;
  username?: string;
}

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new ApiError(text || res.statusText, res.status);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  login: (username: string, password: string) =>
    request<{ username: string }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<void>("/api/auth/logout", { method: "POST" }),
  session: () => request<SessionResponse>("/api/auth/session"),

  status: () => request<StatusResponse>("/api/status"),
  tracking: () => request<TrackingResponse>("/api/tracking"),
  sources: () => request<SourceView[]>("/api/sources"),
  clients: () => request<ClientView[]>("/api/clients"),
  diagnostics: () => request<DiagnosticCheck[]>("/api/diagnostics"),

  sync: () => request<{ result: string }>("/api/sync", { method: "POST" }),
  step: () => request<{ result: string }>("/api/clock/step", { method: "POST" }),
  restart: () => request<{ result: string }>("/api/service/restart", { method: "POST" }),
};

export { ApiError };
