import { readFileSync } from "node:fs";
import { expect, type APIRequestContext, test } from "@playwright/test";
import {
  COVER_PATH,
  LISTEN_AUDIO_DURATION_SECONDS,
  getListenAudioFixture,
} from "../fixtures/generate";
import { resetScreenshots, ScreenshotRecorder } from "../support/screenshots";

const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:8080";
const stamp = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
const seriesTitle = `Harbour Lights ${stamp}`;
const seriesSlug = `harbour-lights-${stamp}`;
const episodeTitles = [
  `The harbour at dawn ${stamp}`,
  `Tides and tables ${stamp}`,
  `Night watch ${stamp}`,
];
const chapterStarts = [0, 30, 60, 90];
const chapterTitles = [
  ["Opening bell", "Harbour at dawn", "Deep current", "Finale"],
  ["First light", "Tide tables", "The long swell", "Sign-off"],
  ["Night watch", "Fog signal", "Sea glass", "Last call"],
];

interface SeriesBody {
  id: string;
  slug: string;
}

interface EpisodeBody {
  id: string;
  number: number;
  title: string;
  audio: { duration_seconds: number } | null;
}

interface ProgressRow {
  episode_id: string;
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

/** Escapes a literal string for use inside a RegExp. */
function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/** Seeds one series with a cover, three episodes, chapters and two published episodes. */
async function seedSeries(
  api: APIRequestContext,
): Promise<{ series: SeriesBody; episodes: EpisodeBody[] }> {
  // The stack may hold the user's real listening progress: never touch rows
  // that do not belong to the episodes seeded by this run.
  const seriesResponse = await api.post("/api/v1/series", {
    data: {
      title: seriesTitle,
      slug: seriesSlug,
      author: "Mira Vale",
      description: "Field recordings from a working harbour, recorded at first light.",
      accent_color: "#F2A541",
    },
  });
  expect(seriesResponse.status()).toBe(201);
  const series = (await seriesResponse.json()) as SeriesBody;

  const coverResponse = await api.put(`/api/v1/series/${series.id}/cover`, {
    multipart: {
      file: {
        name: "cover.png",
        mimeType: "image/png",
        buffer: readFileSync(COVER_PATH),
      },
    },
  });
  expect(coverResponse.status()).toBe(200);

  const audio = getListenAudioFixture();
  const audioBuffer = readFileSync(audio.path);
  const audioType = audio.name.endsWith(".mp3") ? "audio/mpeg" : "audio/wav";
  const episodes: EpisodeBody[] = [];

  for (let index = 0; index < 3; index += 1) {
    const created = await api.post(`/api/v1/series/${series.id}/episodes`, {
      data: {
        number: index + 1,
        title: episodeTitles[index],
        description: `Episode ${index + 1} of the harbour sessions.`,
      },
    });
    expect(created.status()).toBe(201);
    const episode = (await created.json()) as EpisodeBody;

    const uploaded = await api.put(`/api/v1/episodes/${episode.id}/audio`, {
      multipart: {
        duration_seconds: String(LISTEN_AUDIO_DURATION_SECONDS),
        file: { name: audio.name, mimeType: audioType, buffer: audioBuffer },
      },
    });
    if (uploaded.status() !== 200) {
      throw new Error(`audio upload failed: ${uploaded.status()} ${await uploaded.text()}`);
    }
    const withAudio = (await uploaded.json()) as EpisodeBody;
    expect(withAudio.audio?.duration_seconds).toBe(LISTEN_AUDIO_DURATION_SECONDS);

    const chapters = await api.put(`/api/v1/episodes/${episode.id}/chapters`, {
      data: {
        version: "1.2.0",
        chapters: chapterTitles[index].map((title, chapterIndex) => ({
          startTime: chapterStarts[chapterIndex],
          endTime: chapterStarts[chapterIndex] + 30,
          title,
          toc: true,
        })),
      },
    });
    expect(chapters.status()).toBe(200);
    episodes.push(episode);
  }

  for (const episode of episodes.slice(0, 2)) {
    const published = await api.post(`/api/v1/episodes/${episode.id}/publish`);
    expect(published.status()).toBe(200);
  }

  return { series, episodes };
}

/** Reads the listener progress row of one episode, if any. */
async function readProgress(
  api: APIRequestContext,
  episodeId: string,
): Promise<ProgressRow | undefined> {
  const response = await api.get("/api/v1/listening/progress");
  expect(response.status()).toBe(200);
  const body = (await response.json()) as { progress: ProgressRow[] };
  return body.progress.find((row) => row.episode_id === episodeId);
}

test("listener journey", async ({ page, request }) => {
  const shots = new ScreenshotRecorder(page, "listener", 2);
  recorder = shots;

  const { series, episodes } = await seedSeries(request);
  const firstEpisode = episodes[0];
  const firstChapter = chapterTitles[0][0];
  const thirdChapter = chapterTitles[0][2];
  const fourthChapter = chapterTitles[0][3];

  await test.step("open the library", async () => {
    await page.goto("/");
    await expect(page.getByRole("heading", { level: 1, name: "Listen" })).toBeVisible();
    await expect(
      page.getByRole("link", { name: new RegExp(escapeRegExp(seriesTitle)) }),
    ).toBeVisible();
    // The local listener is shared by the whole stack, so only the seeded
    // series is guaranteed to have no saved position.
    await expect(
      page.getByRole("button", { name: new RegExp(`^Resume ${escapeRegExp(seriesTitle)}`) }),
    ).toHaveCount(0);

    await shots.capture(
      "library",
      "Listener library",
      "Mobile library showing the seeded series; it has no Continue listening entry because it was never played.",
    );
  });

  await test.step("open the series", async () => {
    await page.getByRole("link", { name: new RegExp(escapeRegExp(seriesTitle)) }).click();
    await expect(page).toHaveURL(/\/series\/[0-9a-f-]{36}$/);
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();

    const episodeList = page.getByRole("list", { name: `Episodes of ${seriesTitle}` });
    await expect(episodeList.getByRole("listitem")).toHaveCount(2);
    await expect(page.getByText(`/feeds/${seriesSlug}.xml`)).toBeVisible();
    await expect(page.getByText(episodeTitles[2])).toHaveCount(0);

    await shots.capture(
      "series",
      "Series detail",
      "Series page with the cover, the absolute RSS feed URL and the two published episodes; the draft episode is absent.",
    );
  });

  await test.step("play episode 1", async () => {
    await page.getByRole("button", { name: `Play episode 1: ${firstEpisode.title}` }).click();
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
    await page.getByLabel(`Open player: ${firstEpisode.title}`).click();
    await expect(page).toHaveURL(/\/play$/);
    await expect(page.getByText("Chapter 1 of 4", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: firstChapter })).toBeVisible();

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
    await page.getByRole("button", { name: `Chapter 3: ${thirdChapter}` }).click();
    await expect(page.getByText("Chapter 3 of 4", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: thirdChapter })).toBeVisible();

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
    await expect(page.getByRole("heading", { level: 2, name: fourthChapter })).toBeVisible();

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
      .poll(async () => (await readProgress(request, firstEpisode.id))?.position_seconds ?? 0, {
        timeout: 20_000,
      })
      .toBeGreaterThan(0);

    await shots.capture(
      "paused",
      "Playback paused",
      "Playback paused; the position was saved to GET /api/v1/listening/progress.",
    );
  });

  const savedPosition = (await readProgress(request, firstEpisode.id))?.position_seconds ?? 0;
  expect(savedPosition).toBeGreaterThan(chapterStarts[3]);

  await test.step("reload the library", async () => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Continue listening" })).toBeVisible();

    const continueCard = page.getByRole("button", {
      name: `Resume ${seriesTitle} episode 1: ${firstEpisode.title}`,
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
        name: `Resume ${seriesTitle} episode 1: ${firstEpisode.title}`,
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
    await page.goto(`/series/${series.id}`);
    await page.getByRole("button", { name: "More actions for episode 1" }).click();
    await page.getByRole("menuitem", { name: "Mark as played" }).click();

    const firstRow = page
      .getByRole("list", { name: `Episodes of ${seriesTitle}` })
      .getByRole("listitem")
      .filter({ hasText: firstEpisode.title });
    await expect(firstRow.getByText("Played", { exact: true })).toBeVisible();

    await shots.capture(
      "mark-played",
      "Episode marked as played",
      "The series overflow menu marked episode 1 as played and its row now shows Played.",
    );
  });

  await test.step("fetch the RSS feed", async () => {
    const origin = new URL(baseURL).origin;
    const feedResponse = await request.get(`/feeds/${seriesSlug}.xml`);
    expect(feedResponse.status()).toBe(200);
    const feed = await feedResponse.text();

    expect((feed.match(/<item>/g) ?? []).length).toBe(2);
    expect(feed).toContain(episodeTitles[0]);
    expect(feed).toContain(episodeTitles[1]);
    expect(feed).not.toContain(episodeTitles[2]);

    for (const episode of episodes.slice(0, 2)) {
      expect(feed).toContain(
        `<enclosure url="${origin}/api/v1/episodes/${episode.id}/audio"`,
      );
      expect(feed).toContain(
        `<podcast:chapters url="${origin}/api/v1/episodes/${episode.id}/chapters"`,
      );
    }

    await page.goto(`/feeds/${seriesSlug}.xml`);
    await expect(page.locator("body")).toContainText(`<rss`);
    await expect(page.locator("body")).toContainText(episodeTitles[0]);

    await shots.capture(
      "rss-feed",
      "RSS feed",
      "The series feed fetched from the browser returns two items with absolute enclosure and podcast:chapters URLs; the draft episode is absent.",
    );
  });

  await test.step("check the desktop two-column player", async () => {
    await page.goto(`/series/${series.id}`);
    await page.getByRole("button", { name: firstEpisode.title, exact: true }).click();
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

  // Remove the seeded catalog data so the shared stack stays tidy.
  for (const episode of episodes) {
    await request.delete(`/api/v1/listening/progress/${episode.id}`);
    await request.delete(`/api/v1/episodes/${episode.id}`);
  }
  await request.delete(`/api/v1/series/${series.id}`);
});
