import { API_BASE, ApiError, parseErrorResponse } from "./errors";
import type {
  ContinueResponse,
  Episode,
  EpisodeListResponse,
  EpisodeStatus,
  Progress,
  ProgressListResponse,
  Series,
  SeriesListResponse,
} from "./types";

export { ApiError, errorMessage, isAbortError, isApiError, isNotFound } from "./errors";

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

function episodePath(id: string): string {
  return `/episodes/${encodeURIComponent(id)}`;
}

function progressPath(id: string): string {
  return `/listening/progress/${encodeURIComponent(id)}`;
}

function progressBody(positionSeconds: number, completed?: boolean): string {
  return JSON.stringify(
    completed === undefined
      ? { position_seconds: positionSeconds }
      : { position_seconds: positionSeconds, completed },
  );
}

export const api = {
  listSeries(): Promise<SeriesListResponse> {
    return request<SeriesListResponse>("/series");
  },

  getSeries(id: string): Promise<Series> {
    return request<Series>(`/series/${encodeURIComponent(id)}`);
  },

  listEpisodes(seriesId: string, status?: EpisodeStatus): Promise<EpisodeListResponse> {
    const suffix = status ? `?status=${encodeURIComponent(status)}` : "";
    return request<EpisodeListResponse>(
      `/series/${encodeURIComponent(seriesId)}/episodes${suffix}`,
    );
  },

  getEpisode(id: string): Promise<Episode> {
    return request<Episode>(episodePath(id));
  },

  listProgress(): Promise<ProgressListResponse> {
    return request<ProgressListResponse>("/listening/progress");
  },

  continueListening(limit = 5): Promise<ContinueResponse> {
    return request<ContinueResponse>(`/listening/continue?limit=${limit}`);
  },

  recordProgress(episodeId: string, positionSeconds: number): Promise<Progress> {
    return request<Progress>(progressPath(episodeId), {
      method: "PUT",
      body: progressBody(positionSeconds),
    });
  },

  markPlayed(episodeId: string): Promise<Progress> {
    return request<Progress>(`${progressPath(episodeId)}/played`, { method: "POST" });
  },

  markUnplayed(episodeId: string): Promise<void> {
    return request<void>(progressPath(episodeId), { method: "DELETE" });
  },
};

/**
 * Saves the last known position when the page is being unloaded. `keepalive`
 * lets the browser finish the request after the document is gone.
 */
export function saveProgressOnUnload(episodeId: string, positionSeconds: number): void {
  if (!(positionSeconds > 0)) return;
  try {
    void fetch(`${API_BASE}${progressPath(episodeId)}`, {
      method: "PUT",
      keepalive: true,
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: progressBody(positionSeconds),
    }).catch(() => undefined);
  } catch {
    // Nothing sensible to do while the document is unloading.
  }
}
