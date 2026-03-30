import type {
  RecordDetails,
  RecordFilter,
  RecordListItem,
  RecordUpsertInput,
  SessionState,
  SyncResult,
} from "../types";

type BackendAPI = {
  SessionService?: {
    GetSessionState: () => Promise<SessionState>;
    Login: (email: string, password: string) => Promise<SessionState>;
    Register: (email: string, password: string) => Promise<void>;
    Logout: () => Promise<SessionState>;
  };
  RecordsService?: {
    ListRecords: (filter: RecordFilter) => Promise<RecordListItem[]>;
    GetRecord: (id: number) => Promise<RecordDetails>;
    CreateRecord: (input: RecordUpsertInput) => Promise<RecordDetails>;
    UpdateRecord: (input: RecordUpsertInput) => Promise<RecordDetails>;
    DeleteRecord: (id: number) => Promise<void>;
    SyncNow: () => Promise<SyncResult>;
  };
  BinaryService?: {
    PickFileForUpload: () => Promise<string>;
    SaveBinaryAs: (recordId: number) => Promise<string>;
    DownloadBinary: (recordId: number, savePath: string) => Promise<string>;
  };
};

declare global {
  interface Window {
    go?: {
      backend?: BackendAPI;
    };
  }
}

function backend() {
  const value = window.go?.backend;
  if (!value) {
    throw new Error("Wails backend is unavailable");
  }
  return value;
}

export async function getSessionState() {
  return call(backend().SessionService?.GetSessionState);
}

export async function login(email: string, password: string) {
  return call(backend().SessionService?.Login, email, password);
}

export async function register(email: string, password: string) {
  return call(backend().SessionService?.Register, email, password);
}

export async function logout() {
  return call(backend().SessionService?.Logout);
}

export async function listRecords(filter: RecordFilter) {
  return call(backend().RecordsService?.ListRecords, filter);
}

export async function getRecord(id: number) {
  return call(backend().RecordsService?.GetRecord, id);
}

export async function createRecord(input: RecordUpsertInput) {
  return call(backend().RecordsService?.CreateRecord, input);
}

export async function updateRecord(input: RecordUpsertInput) {
  return call(backend().RecordsService?.UpdateRecord, input);
}

export async function deleteRecord(id: number) {
  return call(backend().RecordsService?.DeleteRecord, id);
}

export async function syncNow() {
  return call(backend().RecordsService?.SyncNow);
}

export async function pickFileForUpload() {
  return call(backend().BinaryService?.PickFileForUpload);
}

export async function saveBinaryAs(recordId: number) {
  return call(backend().BinaryService?.SaveBinaryAs, recordId);
}

export async function downloadBinaryToPath(recordId: number, savePath: string) {
  return call(backend().BinaryService?.DownloadBinary, recordId, savePath);
}

async function call<TArgs extends unknown[], TResult>(
  fn: ((...args: TArgs) => Promise<TResult>) | undefined,
  ...args: TArgs
) {
  if (!fn) {
    throw new Error("Requested backend method is unavailable");
  }
  return fn(...args);
}
