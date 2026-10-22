import { API_BASE, ApiError, parseErrorResponse } from "./errors";
import type {
  Episode,
  EpisodeInput,
  EpisodeListResponse,
  EpisodeStatus,
  Series,
  SeriesInput,
  SeriesListResponse,
  UploadProgress,
} from "./types";
import { uploadFile } from "./upload";

export {
  ApiError,
  errorMessage,
  isAbortError,
  isApiError,
  isVersionConflict,
} from "./errors";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        ...(init.body === undefined ? {} : { "Content-Type": "application/json" }),
        ...(init.headers as Record<string, string> | undefined),
      },
    });
  } catch {
    throw new ApiError(
      0,
      "NETWORK_ERROR",
      "Could not reach the server. Check that the API is running.",
    );
  }

  if (!response.ok) throw await parseErrorResponse(response);
  if (response.status === 204) return undefined as T;

  const text = await response.text();
  if (text.length === 0) return undefined as T;
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new ApiError(
      response.status,
      "BAD_RESPONSE",
      "The server returned an unexpected response.",
    );
  }
}

function seriesPath(id: string): string {
  return `/series/${encodeURIComponent(id)}`;
}

function episodePath(id: string): string {
  return `/episodes/${encodeURIComponent(id)}`;
}

export const api = {
  listSeries(query = ""): Promise<SeriesListResponse> {
    const suffix = query.trim() ? `?q=${encodeURIComponent(query.trim())}` : "";
    return request<SeriesListResponse>(`/series${suffix}`);
  },

  createSeries(input: SeriesInput): Promise<Series> {
    return request<Series>("/series", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  getSeries(id: string): Promise<Series> {
    return request<Series>(seriesPath(id));
  },

  updateSeries(id: string, input: SeriesInput): Promise<Series> {
    return request<Series>(seriesPath(id), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  deleteSeries(id: string): Promise<void> {
    return request<void>(seriesPath(id), { method: "DELETE" });
  },

  uploadCover(id: string, file: File): Promise<Series> {
    return uploadFile<Series>({ path: `${seriesPath(id)}/cover`, file });
  },

  removeCover(id: string): Promise<Series> {
    return request<Series>(`${seriesPath(id)}/cover`, { method: "DELETE" });
  },

  listEpisodes(seriesId: string, status?: EpisodeStatus): Promise<EpisodeListResponse> {
    const suffix = status ? `?status=${encodeURIComponent(status)}` : "";
    return request<EpisodeListResponse>(`${seriesPath(seriesId)}/episodes${suffix}`);
  },

  createEpisode(seriesId: string, input: EpisodeInput): Promise<Episode> {
    return request<Episode>(`${seriesPath(seriesId)}/episodes`, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  getEpisode(id: string): Promise<Episode> {
    return request<Episode>(episodePath(id));
  },

  updateEpisode(id: string, input: EpisodeInput): Promise<Episode> {
    return request<Episode>(episodePath(id), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  deleteEpisode(id: string): Promise<void> {
    return request<void>(episodePath(id), { method: "DELETE" });
  },

  uploadAudio(
    id: string,
    file: File,
    durationSeconds: number,
    options: {
      signal?: AbortSignal;
      onProgress?: (progress: UploadProgress) => void;
    } = {},
  ): Promise<Episode> {
    const fields: Record<string, string> = {};
    if (Number.isFinite(durationSeconds) && durationSeconds > 0) {
      fields.duration_seconds = String(durationSeconds);
    }
    return uploadFile<Episode>({
      path: `${episodePath(id)}/audio`,
      file,
      fields,
      signal: options.signal,
      onProgress: options.onProgress,
    });
  },

  setChapters(id: string, rawDocument: string): Promise<Episode> {
    return request<Episode>(`${episodePath(id)}/chapters`, {
      method: "PUT",
      body: rawDocument,
    });
  },

  async getChaptersDocument(id: string): Promise<string> {
    let response: Response;
    try {
      response = await fetch(`${episodePath(id)}/chapters`, {
        headers: { Accept: "application/json+chapters, application/json" },
      });
    } catch {
      throw new ApiError(0, "NETWORK_ERROR", "Could not reach the server.");
    }
    if (!response.ok) throw await parseErrorResponse(response);
    return response.text();
  },

  publishEpisode(id: string): Promise<Episode> {
    return request<Episode>(`${episodePath(id)}/publish`, { method: "POST" });
  },

  unpublishEpisode(id: string): Promise<Episode> {
    return request<Episode>(`${episodePath(id)}/unpublish`, { method: "POST" });
  },
};

export type { UploadProgress };
