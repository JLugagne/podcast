import type { ApiErrorBody } from "./types";

/** The base path of the versioned HTTP API. */
export const API_BASE = "/api/v1";

/**
 * Error returned by the API, or raised by the client for network / protocol
 * failures. `status` is 0 when the request never reached the server.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export function isVersionConflict(error: unknown): boolean {
  return error instanceof ApiError && error.status === 409 && error.code === "VERSION_CONFLICT";
}

export function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === "AbortError";
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error) return error.message;
  return "An unexpected error occurred.";
}

export async function parseErrorResponse(response: Response): Promise<ApiError> {
  let body: Partial<ApiErrorBody> = {};
  try {
    body = (await response.json()) as Partial<ApiErrorBody>;
  } catch {
    // The API usually returns JSON errors; fall back to the status line.
  }
  const code =
    typeof body.code === "string" && body.code.length > 0 ? body.code : `HTTP_${response.status}`;
  const message =
    typeof body.message === "string" && body.message.length > 0
      ? body.message
      : response.statusText || `Request failed with status ${response.status}.`;
  return new ApiError(response.status, code, message);
}
