import type { RecordDetails, RecordFilter, RecordListItem, RecordUpsertInput } from "../types";
import { toListItem, toRecordDetails, type ServerRecordDto } from "../lib/records";
import { currentSession } from "./auth";
import { request } from "./http";
import { uploadBinaryWithKeyVersion } from "./uploads";

type ListResponse = {
  records: ServerRecordDto[];
};

type RecordResponse = {
  record: ServerRecordDto;
};

export async function listRecords(filter: RecordFilter): Promise<RecordListItem[]> {
  const params = new URLSearchParams();
  if (filter !== "all") {
    params.set("type", filter);
  }

  const suffix = params.size > 0 ? `?${params.toString()}` : "";
  const response = await request<ListResponse>(`/records${suffix}`);
  return response.records.map(toListItem);
}

export async function getRecord(id: number): Promise<RecordDetails> {
  const response = await request<RecordResponse>(`/records/${id}`);
  return toRecordDetails(response.record);
}

export async function createRecord(input: RecordUpsertInput): Promise<RecordDetails> {
  const session = currentSession();
  if (!session) {
    throw new Error("not authenticated");
  }

  const payloadVersion =
    input.type === "binary" ? 1 : 0;

  const response = await request<RecordResponse>("/records", {
    method: "POST",
    body: JSON.stringify({
      type: input.type,
      name: input.name,
      metadata: input.metadata,
      device_id: session.deviceId,
      key_version: 0,
      payload_version: payloadVersion,
      ...payloadForRequest(input),
    }),
  });

  if (input.type === "binary" && input.file) {
    await uploadBinaryWithKeyVersion(
      response.record.id,
      input.file,
      response.record.key_version,
    );
  }

  return toRecordDetails(response.record);
}

export async function updateRecord(input: RecordUpsertInput): Promise<RecordDetails> {
  const session = currentSession();
  if (!session) {
    throw new Error("not authenticated");
  }

  const payloadVersion =
    input.type === "binary" && input.file
      ? Math.max(1, input.payloadVersion + 1)
      : input.payloadVersion;

  const response = await request<RecordResponse>(`/records/${input.id}`, {
    method: "PUT",
    body: JSON.stringify({
      name: input.name,
      metadata: input.metadata,
      revision: input.revision + 1,
      device_id: session.deviceId,
      key_version: input.keyVersion,
      payload_version: payloadVersion,
      ...payloadForRequest(input),
    }),
  });

  if (input.type === "binary" && input.file) {
    await uploadBinaryWithKeyVersion(
      response.record.id,
      input.file,
      response.record.key_version,
    );
  }

  return toRecordDetails(response.record);
}

export async function deleteRecord(recordId: number) {
  const session = currentSession();
  if (!session) {
    throw new Error("not authenticated");
  }

  await request<void>(
    `/records/${recordId}`,
    {
      method: "DELETE",
      body: JSON.stringify({
        device_id: session.deviceId,
      }),
    },
    { responseType: "void" },
  );
}

function payloadForRequest(input: RecordUpsertInput) {
  switch (input.type) {
    case "login":
      return {
        login: {
          login: input.login,
          password: input.password,
        },
      };
    case "text":
      return {
        text: {
          content: input.content,
        },
      };
    case "card":
      return {
        card: {
          number: input.number,
          holder_name: input.holder,
          expiry_date: input.expiry,
          cvv: input.cvv,
        },
      };
    case "binary":
      return {
        binary: {},
      };
    default:
      throw new Error("unsupported record type");
  }
}
