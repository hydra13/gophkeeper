export type RecordType = "login" | "text" | "binary" | "card";
export type RecordFilter = "all" | RecordType;

export type SessionState = {
  authenticated: boolean;
  email: string;
  userId: number;
  deviceId: string;
  appName: string;
  version: string;
  serverAddress: string;
};

export type RecordPayload = {
  login: string;
  password: string;
  content: string;
  number: string;
  holder: string;
  expiry: string;
  cvv: string;
  binarySize: number;
};

export type RecordListItem = {
  id: number;
  type: RecordType;
  name: string;
  metadata: string;
  metadataPreview: string;
  revision: number;
  payloadVersion: number;
  payload: RecordPayload;
};

export type RecordDetails = {
  id: number;
  type: RecordType;
  name: string;
  metadata: string;
  revision: number;
  deviceId: string;
  keyVersion: number;
  payloadVersion: number;
  createdAt: string;
  updatedAt: string;
  payload: RecordPayload;
};

export type RecordUpsertInput = {
  id: number;
  type: RecordType;
  name: string;
  metadata: string;
  login: string;
  password: string;
  content: string;
  number: string;
  holder: string;
  expiry: string;
  cvv: string;
  file: File | null;
  revision: number;
  keyVersion: number;
  payloadVersion: number;
};

export type SyncResult = {
  message: string;
};

export type StoredSession = {
  accessToken: string;
  refreshToken: string;
  email: string;
  userId: number;
  deviceId: string;
  lastSyncRevision: number;
};
