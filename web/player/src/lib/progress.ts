import type { Progress } from "../api/types";
import { readStorage, writeStorage } from "./storage";

const PROGRESS_KEY = "progress";

type ProgressMap = Record<string, Progress>;

function readMap(): ProgressMap {
  const raw = readStorage(PROGRESS_KEY);
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    return parsed as ProgressMap;
  } catch {
    return {};
  }
}

function writeMap(map: ProgressMap): void {
  writeStorage(PROGRESS_KEY, JSON.stringify(map));
}

/** Every saved position, most recently updated first. */
export function listProgress(): Progress[] {
  return Object.values(readMap()).sort((left, right) =>
    right.updated_at.localeCompare(left.updated_at),
  );
}

export function getProgress(episodeId: string): Progress | undefined {
  return readMap()[episodeId];
}

/**
 * Persists a position in this browser. A position within two seconds of the
 * end marks the episode completed.
 */
export function recordProgress(
  episodeId: string,
  positionSeconds: number,
  durationSeconds: number,
  completed?: boolean,
): void {
  const position = Number.isFinite(positionSeconds) ? Math.max(0, positionSeconds) : 0;
  const isCompleted = completed ?? (durationSeconds > 0 && position >= durationSeconds - 2);
  if (position <= 0 && !isCompleted) {
    markUnplayed(episodeId);
    return;
  }
  const map = readMap();
  map[episodeId] = {
    episode_id: episodeId,
    position_seconds: position,
    completed: isCompleted,
    updated_at: new Date().toISOString(),
  };
  writeMap(map);
}

/** Marks an episode as fully played without changing the saved position. */
export function markPlayed(episodeId: string, durationSeconds: number): void {
  const map = readMap();
  map[episodeId] = {
    episode_id: episodeId,
    position_seconds: Math.max(0, durationSeconds),
    completed: true,
    updated_at: new Date().toISOString(),
  };
  writeMap(map);
}

/** Forgets every saved position for an episode. */
export function markUnplayed(episodeId: string): void {
  const map = readMap();
  if (!(episodeId in map)) return;
  delete map[episodeId];
  writeMap(map);
}
