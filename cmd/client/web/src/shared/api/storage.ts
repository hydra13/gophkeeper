import type { SessionState, StoredSession } from "../types";
import { defaultSessionState } from "../lib/records";

const sessionKey = "gophkeeper.web.session";
const deviceKey = "gophkeeper.web.deviceId";

type JwtClaims = {
  uid?: number;
  exp?: number;
};

export function getStoredSession(): StoredSession | null {
  const raw = window.localStorage.getItem(sessionKey);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as StoredSession;
  } catch {
    clearStoredSession();
    return null;
  }
}

export function saveStoredSession(session: StoredSession) {
  window.localStorage.setItem(sessionKey, JSON.stringify(session));
}

export function clearStoredSession() {
  window.localStorage.removeItem(sessionKey);
}

export function ensureDeviceId() {
  const existing = window.localStorage.getItem(deviceKey);
  if (existing) {
    return existing;
  }

  const generated =
    "web-" +
    (typeof crypto.randomUUID === "function"
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(16).slice(2)}`);
  window.localStorage.setItem(deviceKey, generated);
  return generated;
}

export function decodeJwtClaims(token: string): JwtClaims {
  const parts = token.split(".");
  if (parts.length < 2) {
    throw new Error("invalid access token");
  }

  const normalized = parts[1].replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  const json = window.atob(padded);
  return JSON.parse(json) as JwtClaims;
}

export function userIdFromAccessToken(token: string) {
  const claims = decodeJwtClaims(token);
  if (!claims.uid || claims.uid <= 0) {
    throw new Error("access token does not contain user id");
  }
  return claims.uid;
}

export function isAccessTokenExpired(token: string) {
  try {
    const claims = decodeJwtClaims(token);
    if (!claims.exp) {
      return false;
    }
    const now = Math.floor(Date.now() / 1000);
    return claims.exp <= now+10;
  } catch {
    return true;
  }
}

export function toSessionState(session: StoredSession | null): SessionState {
  const base = defaultSessionState();
  if (!session) {
    return {
      ...base,
      deviceId: ensureDeviceId(),
    };
  }

  return {
    ...base,
    authenticated: true,
    email: session.email,
    userId: session.userId,
    deviceId: session.deviceId,
  };
}
