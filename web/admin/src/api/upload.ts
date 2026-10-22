import { API_BASE, ApiError } from "./errors";
import type { UploadProgress } from "./types";

export interface UploadFileOptions {
  path: string;
  file: File;
  /** Multipart field name for the file part. */
  field?: string;
  fields?: Record<string, string>;
  method?: "POST" | "PUT";
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
}

export interface UploadRequest {
  method: "POST" | "PUT";
  url: string;
  field: string;
  file: File;
  fields: Record<string, string>;
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
}

export interface UploadResponse {
  status: number;
  body: string;
}

export type UploadTransport = (request: UploadRequest) => Promise<UploadResponse>;

interface ApiErrorBody {
  code?: string;
  message?: string;
}

/**
 * Default transport: multipart/form-data over XMLHttpRequest, which (unlike
 * fetch) reports upload progress and can be aborted mid-flight.
 */
export const xhrUploadTransport: UploadTransport = ({
  method,
  url,
  field,
  file,
  fields,
  signal,
  onProgress,
}) =>
  new Promise<UploadResponse>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException("Upload cancelled", "AbortError"));
      return;
    }

    const xhr = new XMLHttpRequest();
    xhr.open(method, url);
    xhr.setRequestHeader("Accept", "application/json");
    xhr.responseType = "text";

    const abort = () => xhr.abort();
    signal?.addEventListener("abort", abort, { once: true });

    const cleanup = () => {
      signal?.removeEventListener("abort", abort);
    };

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable || event.total === 0) return;
      const percent = Math.min(100, Math.round((event.loaded / event.total) * 100));
      onProgress?.({ percent, loaded: event.loaded, total: event.total });
    };

    xhr.onload = () => {
      cleanup();
      resolve({ status: xhr.status, body: xhr.responseText });
    };

    xhr.onerror = () => {
      cleanup();
      reject(new ApiError(0, "NETWORK_ERROR", "The upload failed because of a network error."));
    };

    xhr.onabort = () => {
      cleanup();
      reject(new DOMException("Upload cancelled", "AbortError"));
    };

    xhr.send(buildUploadForm(field, file, fields));
  });

/**
 * Builds the multipart body with every text field BEFORE the file part: the
 * server streams the file and never reads parts that follow it.
 */
export function buildUploadForm(
  field: string,
  file: File,
  fields: Record<string, string>,
): FormData {
  const form = new FormData();
  for (const [key, value] of Object.entries(fields)) {
    form.append(key, value);
  }
  form.append(field, file, file.name);
  return form;
}

let activeTransport: UploadTransport = xhrUploadTransport;

/** Replaces the upload transport, or restores the XHR transport with `null`. */
export function setUploadTransport(transport: UploadTransport | null): void {
  activeTransport = transport ?? xhrUploadTransport;
}

export async function uploadFile<T>(options: UploadFileOptions): Promise<T> {
  const response = await activeTransport({
    method: options.method ?? "PUT",
    url: `${API_BASE}${options.path}`,
    field: options.field ?? "file",
    file: options.file,
    fields: options.fields ?? {},
    signal: options.signal,
    onProgress: options.onProgress,
  });

  if (response.status < 200 || response.status >= 300) {
    let code = `HTTP_${response.status}`;
    let message = response.status
      ? `Upload failed with status ${response.status}.`
      : "Upload failed.";
    try {
      const body = JSON.parse(response.body) as ApiErrorBody;
      if (typeof body.code === "string" && body.code.length > 0) code = body.code;
      if (typeof body.message === "string" && body.message.length > 0) message = body.message;
    } catch {
      // Keep the status-based fallback.
    }
    throw new ApiError(response.status, code, message);
  }

  if (response.body.length === 0) return undefined as T;
  try {
    return JSON.parse(response.body) as T;
  } catch {
    throw new ApiError(
      response.status,
      "BAD_RESPONSE",
      "The server returned an unexpected response.",
    );
  }
}
