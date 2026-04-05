import type { SyncResult } from "../types";
import { currentSession } from "./auth";
import { request } from "./http";
import { saveStoredSession } from "./storage";

type PullResponse = {
  changes: Array<unknown>;
  next_revision: number;
  has_more: boolean;
};

export async function syncNow(): Promise<SyncResult> {
  const session = currentSession();
  if (!session) {
    throw new Error("not authenticated");
  }

  const response = await request<PullResponse>("/sync/pull", {
    method: "POST",
    body: JSON.stringify({
      user_id: session.userId,
      device_id: session.deviceId,
      since_revision: session.lastSyncRevision,
      limit: 500,
    }),
  });

  saveStoredSession({
    ...session,
    lastSyncRevision: Math.max(session.lastSyncRevision, response.next_revision),
  });

  const suffix = response.has_more ? ", more changes remain on server" : "";
  return {
    message: `sync completed, received ${response.changes.length} change(s)${suffix}`,
  };
}
