import type { StoredSession } from "../types";
import {
  clearStoredSession,
  getStoredSession,
  saveStoredSession,
  userIdFromAccessToken,
} from "./storage";

type RequestOptions = {
  auth?: boolean;
  responseType?: "json" | "blob" | "void";
  retryOnAuth?: boolean;
};

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export async function request<T>(
  path: string,
  init: RequestInit = {},
  options: RequestOptions = {},
): Promise<T> {
  const responseType = options.responseType ?? "json";
  const auth = options.auth ?? true;
  const retryOnAuth = options.retryOnAuth ?? true;

  const session = getStoredSession();
  const headers = new Headers(init.headers);

  if (responseType === "json" && !headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json");
  }
  if (auth && session?.accessToken) {
    headers.set("Authorization", `Bearer ${session.accessToken}`);
  }

  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers,
  });

  if (response.status === 401 && auth && retryOnAuth && session?.refreshToken) {
    const refreshed = await refreshTokens(session);
    if (refreshed) {
      return request<T>(path, init, { ...options, retryOnAuth: false });
    }
  }

  if (!response.ok) {
    throw new Error(await readError(response));
  }

  if (responseType === "void") {
    return undefined as T;
  }
  if (responseType === "blob") {
    return (await response.blob()) as T;
  }

  return (await response.json()) as T;
}

async function refreshTokens(session: StoredSession) {
  const response = await fetch(`${apiBaseUrl}/auth/refresh`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      refresh_token: session.refreshToken,
    }),
  });

  if (!response.ok) {
    clearStoredSession();
    return false;
  }

  const payload = (await response.json()) as {
    access_token: string;
    refresh_token: string;
  };

  saveStoredSession({
    ...session,
    accessToken: payload.access_token,
    refreshToken: payload.refresh_token,
    userId: userIdFromAccessToken(payload.access_token),
  });

  return true;
}

async function readError(response: Response) {
  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    const payload = (await response.json()) as { error?: string; message?: string };
    return payload.error ?? payload.message ?? `Request failed with status ${response.status}`;
  }

  const text = await response.text();
  return text || `Request failed with status ${response.status}`;
}
