import { request } from "./http";
import { currentSession } from "./auth";

const binaryChunkSize = 64 * 1024;

type CreateUploadResponse = {
  upload_id: number;
};

export async function uploadBinary(recordId: number, file: File) {
  const session = currentSession();
  if (!session) {
    throw new Error("not authenticated");
  }
  throw new Error("uploadBinary(recordId, file) should not be called without keyVersion");
}

export async function uploadBinaryWithKeyVersion(
  recordId: number,
  file: File,
  keyVersion: number,
) {
  const session = currentSession();
  if (!session) {
    throw new Error("not authenticated");
  }

  const totalChunks = Math.ceil(file.size / binaryChunkSize);
  const create = await request<CreateUploadResponse>("/uploads", {
    method: "POST",
    body: JSON.stringify({
      user_id: session.userId,
      record_id: recordId,
      total_chunks: totalChunks,
      chunk_size: binaryChunkSize,
      total_size: file.size,
      key_version: keyVersion,
    }),
  });

  for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex += 1) {
    const start = chunkIndex * binaryChunkSize;
    const end = Math.min(start + binaryChunkSize, file.size);
    const chunk = await file.slice(start, end).arrayBuffer();

    await request(
      `/uploads/${create.upload_id}/chunks`,
      {
        method: "POST",
        body: JSON.stringify({
          chunk_index: chunkIndex,
          data: arrayBufferToBase64(chunk),
        }),
      },
      { responseType: "json" },
    );
  }
}

export async function downloadBinary(recordId: number, fileName: string) {
  const blob = await request<Blob>(
    `/records/${recordId}/binary`,
    {
      method: "GET",
    },
    { responseType: "blob" },
  );

  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName || `record-${recordId}.bin`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

function arrayBufferToBase64(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer);
  let binary = "";

  for (let index = 0; index < bytes.length; index += 1) {
    binary += String.fromCharCode(bytes[index]);
  }

  return window.btoa(binary);
}
