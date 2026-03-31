import type {
  RecordDetails,
  RecordListItem,
  RecordPayload,
  RecordType,
  SessionState,
} from "../types";

export const recordFilters: Array<{ label: string; value: RecordType | "all" }> = [
  { label: "All", value: "all" },
  { label: "Login", value: "login" },
  { label: "Text", value: "text" },
  { label: "Binary", value: "binary" },
  { label: "Card", value: "card" },
];

export type ServerRecordDto = {
  id: number;
  user_id: number;
  type: RecordType;
  name: string;
  metadata?: string;
  revision: number;
  device_id: string;
  key_version: number;
  payload_version?: number;
  created_at: string;
  updated_at: string;
  payload:
    | { login: string; password: string }
    | { content: string }
    | { number: string; holder_name: string; expiry_date: string; cvv: string }
    | Record<string, never>;
};

export function defaultSessionState(): SessionState {
  return {
    authenticated: false,
    email: "",
    userId: 0,
    deviceId: "",
    appName: "GophKeeper Web",
    version: "dev",
    serverAddress: import.meta.env.VITE_API_BASE_URL ?? "/api/v1",
  };
}

export function metadataPreview(metadata: string, max: number) {
  if (!metadata) {
    return "";
  }

  const [firstLine] = metadata.split("\n");
  if (max > 0 && firstLine.length > max) {
    return `${firstLine.slice(0, max)}...`;
  }
  return firstLine;
}

export function toListItem(record: ServerRecordDto): RecordListItem {
  return {
    id: record.id,
    type: record.type,
    name: record.name,
    metadata: record.metadata ?? "",
    metadataPreview: metadataPreview(record.metadata ?? "", 80),
    revision: record.revision,
    payloadVersion: record.payload_version ?? 0,
    payload: toPayload(record),
  };
}

export function toRecordDetails(record: ServerRecordDto): RecordDetails {
  return {
    id: record.id,
    type: record.type,
    name: record.name,
    metadata: record.metadata ?? "",
    revision: record.revision,
    deviceId: record.device_id,
    keyVersion: record.key_version,
    payloadVersion: record.payload_version ?? 0,
    createdAt: record.created_at,
    updatedAt: record.updated_at,
    payload: toPayload(record),
  };
}

function toPayload(record: ServerRecordDto): RecordPayload {
  switch (record.type) {
    case "login": {
      const payload = record.payload as { login?: string; password?: string };
      return {
        login: payload.login ?? "",
        password: payload.password ?? "",
        content: "",
        number: "",
        holder: "",
        expiry: "",
        cvv: "",
        binarySize: 0,
      };
    }
    case "text": {
      const payload = record.payload as { content?: string };
      return {
        login: "",
        password: "",
        content: payload.content ?? "",
        number: "",
        holder: "",
        expiry: "",
        cvv: "",
        binarySize: 0,
      };
    }
    case "card": {
      const payload = record.payload as {
        number?: string;
        holder_name?: string;
        expiry_date?: string;
        cvv?: string;
      };
      return {
        login: "",
        password: "",
        content: "",
        number: payload.number ?? "",
        holder: payload.holder_name ?? "",
        expiry: payload.expiry_date ?? "",
        cvv: payload.cvv ?? "",
        binarySize: 0,
      };
    }
    case "binary":
    default:
      return {
        login: "",
        password: "",
        content: "",
        number: "",
        holder: "",
        expiry: "",
        cvv: "",
        binarySize: 0,
      };
  }
}
