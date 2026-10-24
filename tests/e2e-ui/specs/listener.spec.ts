import { expect, type Page, test } from "@playwright/test";
import {
  chapterStarts,
  episodes,
  FEED_URL,
  type HarbourHost,
  installHarbourHost,
  seriesTitle,
} from "../fixtures/harbour";
import { resetScreenshots, ScreenshotRecorder } from "../support/screenshots";

const episodeTitles = episodes.map((episode) => episode.title);
const chapterTitles = episodes[0].chapters;

// The slug and episode ids derive from the feed URL and GUIDs; they are read
// back from the saved library once the podcast is added.
let seriesSlug = "";
let episodeId = "";

interface StoredProgress {
  position_seconds: number;
  completed: boolean;
}

let recorder: ScreenshotRecorder | undefined;

test.beforeAll(() => {
  resetScreenshots("listener");
});

test.afterEach(async ({}, testInfo) => {
  if (recorder && testInfo.status !== testInfo.expectedStatus) recorder.failed = true;
});

test.afterAll(() => {
  recorder?.writeManifest();
});

/** Reads the subscriptions saved in this browser's localStorage. */
async function readLibrary(page: Page): Promise<{ slug: string; episodes: { id: string; number: number }[] }[]> {
  return page.evaluate(() => {
    const raw = window.localStorage.getItem("podcast.player.library");
    return raw ? JSON.parse(raw) : [];
  });
}

/** Reads one episode's progress from this browser's localStorage. */
async function readProgress(page: Page, id: string): Promise<StoredProgress | null> {
  return page.evaluate((episode) => {
    const raw = window.localStorage.getItem("podcast.player.progress");
    if (!raw) return null;
    const map = JSON.parse(raw) as Record<string, StoredProgress>;
    return map[episode] ?? null;
  }, id);
}

test("listener journey", async ({ page, context }) => {
  const host: HarbourHost = await installHarbourHost(context);
  const shots = new ScreenshotRecorder(page, "listener", 2);
  recorder = shots;

  await test.step("open an empty library", async () => {
    await page.goto("/");
    await page.evaluate(() => window.localStorage.clear());
    await page.reload();

    await expect(page.getByRole("heading", { level: 1, name: "Listen" })).toBeVisible();
    await expect(page.getByText("Nothing to listen to yet")).toBeVisible();

    await shots.capture(
      "empty-library",
      "Empty library",
      "First launch: the library is empty and offers to add a first podcast; the feed form only appears when asked for.",
    );
  });

  await test.step("add the podcast from its feed URL", async () => {
    await page.getByRole("button", { name: "Add podcast", exact: true }).click();
    await page.getByLabel("Podcast feed URL").fill(FEED_URL);
    await page.getByRole("button", { name: "Add", exact: true }).click();

    await expect(page.getByRole("link", { name: new RegExp(seriesTitle) })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Continue listening" })).toHaveCount(0);

    const library = await readLibrary(page);
    expect(library).toHaveLength(1);
    seriesSlug = library[0].slug;
    episodeId = library[0].episodes.find((episode) => episode.number === 1)?.id ?? "";
    expect(episodeId).not.toBe("");

    await shots.capture(
      "library",
      "Library with a podcast",
      "The podcast added from its feed URL is listed in the library; there is no Continue listening entry before playback.",
    );
  });

  await test.step("open a shared link for a podcast already in the library", async () => {
    await page.goto(`/?feed=${encodeURIComponent(FEED_URL)}`);

    await expect(page.getByText("1 podcast already in your library")).toBeVisible();
    await expect(page.getByRole("link", { name: new RegExp(seriesTitle) })).toBeVisible();
    // The feed list is dropped from the address bar once it has been imported.
    await expect(page).toHaveURL(new RegExp("/$"));
    expect(await readLibrary(page)).toHaveLength(1);
  });

  await test.step("keep the podcast after a reload", async () => {
    await page.reload();
    await expect(page.getByRole("link", { name: new RegExp(seriesTitle) })).toBeVisible();
  });

  await test.step("open the series", async () => {
    await page.getByRole("link", { name: new RegExp(seriesTitle) }).click();
    await expect(page).toHaveURL(new RegExp(`/series/${seriesSlug}$`));
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();

    const episodeList = page.getByRole("list", { name: `Episodes of ${seriesTitle}` });
    await expect(episodeList.getByRole("listitem")).toHaveCount(3);
    await expect(page.getByText(episodeTitles[2])).toBeVisible();

    await shots.capture(
      "series",
      "Series detail",
      "Series page with the cover, the description and the three episodes read from the feed.",
    );
  });

  await test.step("play episode 1", async () => {
    await page.getByRole("button", { name: `Play episode 1: ${episodeTitles[0]}` }).click();
    const miniPlayer = page.getByRole("region", { name: "Mini player" });
    await expect(miniPlayer).toBeVisible();
    await expect(miniPlayer.getByRole("button", { name: "Pause" })).toBeVisible();

    await shots.capture(
      "episode-playing",
      "Episode playing",
      "Episode 1 started from the series list; the mini player is docked at the bottom of the screen.",
    );
  });

  await test.step("open the full player", async () => {
    await page.getByLabel(`Open player: ${episodeTitles[0]}`).click();
    await expect(page).toHaveURL(/\/play$/);
    await expect(page.getByText("Chapter 1 of 4", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: chapterTitles[0] })).toBeVisible();

    const slider = page.getByRole("slider", { name: "Playback position" });
    await expect(slider).toBeVisible();
    await expect
      .poll(async () => Number(await slider.getAttribute("aria-valuenow")), { timeout: 20_000 })
      .toBeGreaterThan(0);

    await shots.capture(
      "full-player",
      "Full player",
      "Full-screen mobile player with the chapter title, the chapter-segmented progress bar and the playback controls.",
    );
  });

  await test.step("jump to chapter 3", async () => {
    await page.getByRole("button", { name: `Chapter 3: ${chapterTitles[2]}` }).click();
    await expect(page.getByText("Chapter 3 of 4", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: chapterTitles[2] })).toBeVisible();

    const slider = page.getByRole("slider", { name: "Playback position" });
    await expect
      .poll(async () => Number(await slider.getAttribute("aria-valuenow")))
      .toBeGreaterThanOrEqual(chapterStarts[2]);

    await shots.capture(
      "chapter-jump",
      "Chapter jump",
      "Jumping to chapter 3 from the chapter list moved playback to 1:00 and updated the chapter label.",
    );
  });

  await test.step("next chapter", async () => {
    await page.getByRole("button", { name: "Next chapter" }).click();
    await expect(page.getByText("Chapter 4 of 4", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: chapterTitles[3] })).toBeVisible();

    const slider = page.getByRole("slider", { name: "Playback position" });
    await expect
      .poll(async () => Number(await slider.getAttribute("aria-valuenow")))
      .toBeGreaterThanOrEqual(chapterStarts[3]);

    await shots.capture(
      "next-chapter",
      "Next chapter",
      "The next chapter button advanced playback to chapter 4 and the segmented bar highlighted its segment.",
    );
  });

  await test.step("change the playback speed", async () => {
    await page.getByRole("button", { name: "Playback speed: 1×" }).click();
    await page.getByRole("menuitemradio", { name: "1.5×" }).click();
    await expect(page.getByRole("button", { name: "Playback speed: 1.5×" })).toBeVisible();

    await shots.capture(
      "speed",
      "Playback speed 1.5x",
      "The speed menu set playback to 1.5x; the trigger now shows the active rate.",
    );
  });

  await test.step("set the sleep timer", async () => {
    await page.getByRole("button", { name: "Sleep timer off" }).click();
    await page.getByRole("menuitemradio", { name: "15 minutes" }).click();
    await expect(page.getByRole("button", { name: /Sleep timer \d+:\d{2}/ })).toBeVisible();

    await shots.capture(
      "sleep-timer",
      "Sleep timer set",
      "A 15 minute sleep timer is running and its countdown is shown on the player.",
    );
  });

  await test.step("pause playback", async () => {
    await page.getByRole("button", { name: "Pause", exact: true }).click();
    await expect(page.getByRole("button", { name: "Play", exact: true })).toBeVisible();

    await expect
      .poll(async () => (await readProgress(page, episodeId))?.position_seconds ?? 0, {
        timeout: 20_000,
      })
      .toBeGreaterThan(0);

    await shots.capture(
      "paused",
      "Playback paused",
      "Playback paused; the position was saved in this browser's localStorage.",
    );
  });

  const savedPosition = (await readProgress(page, episodeId))?.position_seconds ?? 0;
  expect(savedPosition).toBeGreaterThan(chapterStarts[3]);

  await test.step("reload the library", async () => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Continue listening" })).toBeVisible();

    const continueCard = page.getByRole("button", {
      name: `Resume ${seriesTitle} episode 1: ${episodeTitles[0]}`,
    });
    await expect(continueCard).toBeVisible();
    await expect(continueCard).toBeEnabled();
    await expect(continueCard).toContainText(/Chapter 4: /);
    await expect(continueCard).toContainText(/min left/);

    await shots.capture(
      "continue-listening",
      "Continue listening",
      "After a reload the library lists episode 1 under Continue listening with its current chapter and time left.",
    );
  });

  await test.step("resume from the continue card", async () => {
    await page
      .getByRole("button", {
        name: `Resume ${seriesTitle} episode 1: ${episodeTitles[0]}`,
      })
      .click();
    await expect(page).toHaveURL(/\/play$/);

    const slider = page.getByRole("slider", { name: "Playback position" });
    await expect(slider).toBeVisible();
    await expect
      .poll(async () => Number(await slider.getAttribute("aria-valuenow")))
      .toBeGreaterThanOrEqual(Math.floor(savedPosition) - 1);

    await shots.capture(
      "resume",
      "Resume playback",
      "Resuming from the continue card restored the saved position (at most one second behind).",
    );
  });

  await test.step("show the mini player again", async () => {
    await page.getByRole("button", { name: "Close player" }).click();
    const miniPlayer = page.getByRole("region", { name: "Mini player" });
    await expect(miniPlayer).toBeVisible();
    await miniPlayer.getByRole("button", { name: "Pause" }).click();
    await expect(miniPlayer.getByRole("button", { name: "Play" })).toBeVisible();

    await shots.capture(
      "mini-player",
      "Mini player",
      "Closing the full player returns to the library with the mini player docked and paused.",
    );
  });

  await test.step("mark episode 1 as played", async () => {
    await page.goto(`/series/${seriesSlug}`);
    await page.getByRole("button", { name: "More actions for episode 1" }).click();
    await page.getByRole("menuitem", { name: "Mark as played" }).click();

    const firstRow = page
      .getByRole("list", { name: `Episodes of ${seriesTitle}` })
      .getByRole("listitem")
      .filter({ hasText: episodeTitles[0] });
    await expect(firstRow.getByText("Played", { exact: true })).toBeVisible();

    await shots.capture(
      "mark-played",
      "Episode marked as played",
      "The series overflow menu marked episode 1 as played; the state lives in this browser's localStorage.",
    );
  });

  await test.step("download an episode and play it without the network", async () => {
    await page.goto(`/series/${seriesSlug}`);
    await page.getByRole("button", { name: "More actions for episode 2" }).click();
    await page.getByRole("menuitem", { name: "Download for offline" }).click();

    const secondRow = page
      .getByRole("list", { name: `Episodes of ${seriesTitle}` })
      .getByRole("listitem")
      .filter({ hasText: episodeTitles[1] });
    await expect(secondRow.getByText("Downloaded", { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "More actions for episode 2" }).click();
    await expect(page.getByRole("menuitem", { name: "Remove download" })).toBeVisible();
    await page.keyboard.press("Escape");

    // From now on the host no longer serves audio: playback must use the download.
    host.audioBlocked = true;
    const requestsBefore = host.audioRequests;
    await page.getByRole("button", { name: `Play episode 2: ${episodeTitles[1]}` }).click();
    const miniPlayer = page.getByRole("region", { name: "Mini player" });
    await expect(miniPlayer.getByRole("button", { name: "Pause" })).toBeVisible();
    await expect(page.getByText("The audio could not be played")).toHaveCount(0);
    await page.getByLabel(`Open player: ${episodeTitles[1]}`).click();
    const slider = page.getByRole("slider", { name: "Playback position" });
    await expect
      .poll(async () => Number(await slider.getAttribute("aria-valuenow")), { timeout: 20_000 })
      .toBeGreaterThan(0);
    expect(host.audioRequests).toBe(requestsBefore);

    await shots.capture(
      "offline-playback",
      "Offline playback",
      "Episode 2 was downloaded, the host then refused audio requests, and it still plays from the offline copy.",
    );
    host.audioBlocked = false;
  });

  await test.step("restart from the beginning", async () => {
    await page.goto(`/series/${seriesSlug}`);
    await page.getByRole("button", { name: episodeTitles[1], exact: true }).click();
    await expect(page).toHaveURL(/\/play$/);
    await page.getByRole("button", { name: `Chapter 3: ${episodes[1].chapters[2]}` }).click();
    await expect(page.getByText("Chapter 3 of 4", { exact: true })).toBeVisible();
    await expect(page.locator(".chapter-row.is-played").first()).toBeVisible();

    await page.getByRole("button", { name: "Restart from the beginning" }).click();
    await expect(page.getByText("Chapter 1 of 4", { exact: true })).toBeVisible();
    await expect(page.locator(".chapter-row.is-played")).toHaveCount(0);

    await shots.capture(
      "restart",
      "Restart from the beginning",
      "Restart from the beginning put playback back at 0:00 and cleared the played markers on the chapters.",
    );
  });

  await test.step("check the desktop two-column player", async () => {
    await page.goto(`/series/${seriesSlug}`);
    await page.getByRole("button", { name: episodeTitles[0], exact: true }).click();
    await expect(page).toHaveURL(/\/play$/);

    await page.setViewportSize({ width: 1280, height: 800 });
    const playerMain = page.getByRole("region", { name: "Player" });
    const playerSide = page.getByRole("complementary", { name: "Chapters and queue" });
    await expect(playerMain).toBeVisible();
    await expect(playerSide).toBeVisible();

    const mainBox = await playerMain.boundingBox();
    const sideBox = await playerSide.boundingBox();
    expect(mainBox).not.toBeNull();
    expect(sideBox).not.toBeNull();
    expect(sideBox?.x ?? 0).toBeGreaterThanOrEqual((mainBox?.x ?? 0) + (mainBox?.width ?? 0));

    await shots.capture(
      "desktop-player",
      "Desktop two-column player",
      "At 1280x800 the player switches to two columns: artwork and controls on the left, chapters and up next on the right.",
    );
  });
});
