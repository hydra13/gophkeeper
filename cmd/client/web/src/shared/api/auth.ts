import type { SessionState, StoredSession } from "../types";
import { defaultSessionState } from "../lib/records";
import { request } from "./http";
import {
  clearStoredSession,
  ensureDeviceId,
  getStoredSession,
  isAccessTokenExpired,
  saveStoredSession,
  toSessionState,
  userIdFromAccessToken,
} from "./storage";

type LoginResponse = {
  access_token: string;
  refresh_token: string;
};

export async function restoreSession(): Promise<SessionState> {
  const existing = getStoredSession();
  if (!existing) {
    return {
      ...defaultSessionState(),
      deviceId: ensureDeviceId(),
    };
  }

  if (isAccessTokenExpired(existing.accessToken)) {
    try {
      await request<LoginResponse>(
        "/auth/refresh",
        {
          method: "POST",
          body: JSON.stringify({
            refresh_token: existing.refreshToken,
          }),
        },
        { auth: false },
      ).then((payload) => {
        persistSession({
          ...existing,
          accessToken: payload.access_token,
          refreshToken: payload.refresh_token,
          userId: userIdFromAccessToken(payload.access_token),
        });
      });
    } catch {
      clearStoredSession();
      return {
        ...defaultSessionState(),
        deviceId: ensureDeviceId(),
      };
    }
  }

  return toSessionState(getStoredSession());
}

export async function login(email: string, password: string): Promise<SessionState> {
  const deviceId = ensureDeviceId();
  const response = await request<LoginResponse>(
    "/auth/login",
    {
      method: "POST",
      body: JSON.stringify({
        email,
        password,
        device_id: deviceId,
        device_name: "web-client",
        client_type: "web",
      }),
    },
    { auth: false },
  );

  persistSession({
    accessToken: response.access_token,
    refreshToken: response.refresh_token,
    email,
    userId: userIdFromAccessToken(response.access_token),
    deviceId,
    lastSyncRevision: getStoredSession()?.lastSyncRevision ?? 0,
  });

  return toSessionState(getStoredSession());
}

export async function register(email: string, password: string) {
  await request<{ user_id: number }>(
    "/auth/register",
    {
      method: "POST",
      body: JSON.stringify({ email, password }),
    },
    { auth: false },
  );
}

export async function logout(): Promise<SessionState> {
  try {
    await request<void>(
      "/auth/logout",
      {
        method: "POST",
      },
      { responseType: "void" },
    );
  } finally {
    clearStoredSession();
  }

  return {
    ...defaultSessionState(),
    deviceId: ensureDeviceId(),
  };
}

export function currentSession(): StoredSession | null {
  return getStoredSession();
}

function persistSession(session: StoredSession) {
  saveStoredSession(session);
}
