import { EPISODE_VIEW_KEY, readStorage, writeStorage } from "./storage";

/** How a series lists its episodes. */
export type EpisodeView = "list" | "grid";

/** The view chosen by the reader, remembered on this device. */
export function readEpisodeView(): EpisodeView {
  return readStorage(EPISODE_VIEW_KEY) === "grid" ? "grid" : "list";
}

export function writeEpisodeView(view: EpisodeView): void {
  writeStorage(EPISODE_VIEW_KEY, view);
}
