import { readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import {
  COVER_PATH,
  getAudioFixture,
  INVALID_CHAPTERS_PATH,
  VALID_CHAPTERS_PATH,
} from "../fixtures/generate";
import { resetScreenshots, ScreenshotRecorder } from "../support/screenshots";

test.describe.configure({ mode: "serial" });

test.beforeAll(() => {
  resetScreenshots("admin");
});

const stamp = Date.now().toString(36);
const seriesTitle = `Deep Focus Lab ${stamp}`;
const seriesSlug = `deep-focus-lab-${stamp}`;
const initialDescription = "A short show about focus, sound and deep work.";
const updatedDescription =
  "A short show about focus, sound and deep work. Updated by the end-to-end suite.";
const episodeTitle = `Signal and noise ${stamp}`;

let recorder: ScreenshotRecorder | undefined;

test.afterEach(async ({}, testInfo) => {
  if (recorder && testInfo.status !== testInfo.expectedStatus) recorder.failed = true;
});

test.afterAll(() => {
  recorder?.writeManifest();
});

test("main admin journey", async ({ page }) => {
  const shots = new ScreenshotRecorder(page, "admin");
  recorder = shots;
  const audio = getAudioFixture();

  await test.step("open the admin series list", async () => {
    await page.goto("/admin/");
    await expect(page.getByRole("heading", { level: 1, name: "Series" })).toBeVisible();
    await shots.capture(
      "series-list",
      "Series list",
      "Admin landing page with the series list and the New series action.",
    );
  });

  let seriesId = "";

  await test.step("create a series", async () => {
    await page.getByRole("link", { name: "New series" }).first().click();
    await expect(page.getByRole("heading", { level: 1, name: "New series" })).toBeVisible();

    await page.getByLabel("Title").fill(seriesTitle);
    await expect(page.getByLabel("Slug", { exact: true })).toHaveValue(seriesSlug);
    await page.getByLabel("Author", { exact: true }).fill("Jordan Field");
    await page.getByLabel("Description", { exact: true }).fill(initialDescription);
    await page.getByLabel("Accent color", { exact: true }).fill("#FF6B35");

    await shots.capture(
      "series-form-filled",
      "New series form",
      "Series form filled with title, auto-derived slug, author, description and an accent color.",
    );

    await page.getByRole("button", { name: "Create series" }).click();
    await expect(page).toHaveURL(/\/admin\/series\/[0-9a-f-]{36}$/);
    seriesId = new URL(page.url()).pathname.split("/").pop() ?? "";
    expect(seriesId).not.toEqual("");
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();
    await expect(page.getByText(initialDescription)).toBeVisible();
    await expect(page.getByText("No episodes yet")).toBeVisible();

    await shots.capture(
      "series-created",
      "Series detail",
      "Newly created series with its cover placeholder and an empty episodes table.",
    );
  });

  await test.step("upload the series cover", async () => {
    await page.getByRole("link", { name: "Edit series" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Edit series" })).toBeVisible();
    await expect(page.getByText(/^Version \d+$/)).toBeVisible();

    await page.locator("#series-cover-file").setInputFiles(COVER_PATH);
    await expect(page.getByRole("button", { name: "Remove cover" })).toBeVisible({
      timeout: 20_000,
    });
    await expect(
      page.locator('section[aria-labelledby="series-cover-heading"] .cover-tile img'),
    ).toBeVisible();

    await shots.capture(
      "series-cover-uploaded",
      "Series cover uploaded",
      "Edit screen showing the uploaded cover preview and the remove action.",
    );

    await page.getByRole("link", { name: "Cancel" }).click();
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();
    await expect(page.locator(".detail-header .cover-tile img")).toBeVisible();

    await shots.capture(
      "series-detail-cover",
      "Series detail with cover",
      "Series detail page showing the uploaded artwork next to the series metadata.",
    );
  });

  await test.step("create an episode", async () => {
    await page.getByRole("link", { name: "New episode" }).first().click();
    await expect(page.getByRole("heading", { level: 1, name: "New episode" })).toBeVisible();

    await page.getByLabel("Title").fill(episodeTitle);
    await page
      .getByLabel("Description", { exact: true })
      .fill("How attention behaves when the room is quiet.");
    await page.getByLabel("Number").fill("1");

    await shots.capture(
      "new-episode-form",
      "New episode form",
      "Episode creation form with number, title and description.",
    );

    await page.getByRole("button", { name: "Create episode" }).click();
    await expect(page).toHaveURL(/\/admin\/episodes\/[0-9a-f-]{36}$/);
    await expect(page.getByRole("heading", { level: 1, name: episodeTitle })).toBeVisible();
    await expect(page.locator(".page-header").getByText("Draft", { exact: true })).toBeVisible();
    await expect(page.getByText("No audio uploaded yet")).toBeVisible();
    await expect(page.getByText("No chapters saved yet")).toBeVisible();

    await shots.capture(
      "episode-editor-empty",
      "Episode editor",
      "Draft episode editor with empty audio, chapters and publishing panels.",
    );
  });

  const audioPlayer = page.locator("audio.audio-player");

  await test.step("upload the episode audio", async () => {
    await page.locator("#episode-audio-file").setInputFiles(audio.path);
    await expect(page.locator(".file-meta")).toBeVisible({ timeout: 30_000 });
    await expect(page.locator(".file-meta")).toContainText(audio.name);
    await expect(page.locator(".file-meta")).toContainText("0:20", { timeout: 20_000 });
    await expect(
      page
        .locator('section[aria-labelledby="episode-audio-heading"]')
        .getByText("Uploaded", { exact: true }),
    ).toBeVisible();
    await expect(audioPlayer).toHaveCount(1);
    await expect(page.locator(".checklist")).toContainText("Audio uploaded");

    await shots.capture(
      "audio-uploaded",
      "Audio uploaded",
      "Audio panel showing the uploaded file, its size, duration and the inline player.",
    );
  });

  await test.step("reject invalid chapters", async () => {
    const invalid = readFileSync(INVALID_CHAPTERS_PATH, "utf8");
    await page.getByLabel("Chapters JSON", { exact: true }).fill(invalid);

    await expect(page.getByText("Fix the chapters document")).toBeVisible();
    await expect(page.getByText("validation issues found")).toBeVisible();
    await expect(page.getByRole("button", { name: "Save chapters" })).toBeDisabled();

    await shots.capture(
      "chapters-invalid",
      "Invalid chapters rejected",
      "Client-side chapters validation errors with the Save chapters action disabled.",
    );
  });

  await test.step("load valid chapters and preview them", async () => {
    await page.locator("#episode-chapters-file").setInputFiles(VALID_CHAPTERS_PATH);

    await expect(page.getByText("The chapters document is valid")).toBeVisible();
    await expect(
      page.getByText("4 chapters, starting at 0:00:00.", { exact: true }),
    ).toBeVisible();
    await expect(page.getByText("Loaded from valid-chapters.json")).toBeVisible();

    const preview = page.getByRole("table", { name: "Chapter preview" });
    await expect(preview).toBeVisible();
    await expect(preview.locator("tbody tr")).toHaveCount(4);
    await expect(preview).toContainText("Cold open");
    await expect(preview).toContainText("Wrap-up");
    await expect(preview).toContainText("0:00:05");

    await shots.capture(
      "chapters-preview",
      "Chapters preview",
      "Valid chapters document loaded from file with the four-chapter preview table.",
    );
  });

  await test.step("save the chapters", async () => {
    await page.getByRole("button", { name: "Save chapters" }).click();
    await expect(page.getByText("Chapters saved.", { exact: true })).toBeVisible();
    await expect(page.locator(".checklist")).toContainText("4 chapters");

    await shots.capture(
      "chapters-saved",
      "Chapters saved",
      "Chapters persisted on the episode with the success confirmation.",
    );
  });

  await test.step("seek the player from a chapter", async () => {
    await expect
      .poll(async () =>
        audioPlayer.evaluate((element) => (element as HTMLAudioElement).readyState),
      )
      .toBeGreaterThan(0);

    await page.getByRole("button", { name: /Play chapter 2 at 0:00:05/ }).click();
    await expect
      .poll(async () =>
        audioPlayer.evaluate((element) => (element as HTMLAudioElement).currentTime),
      )
      .toBeGreaterThan(4.5);

    await shots.capture(
      "chapter-seek",
      "Chapter seek",
      "Clicking a chapter moved the inline audio player to that chapter start time.",
    );
  });

  await test.step("publish the episode", async () => {
    const checklist = page.locator(".checklist");
    await expect(checklist).toContainText("Audio uploaded");
    await expect(checklist).toContainText("4 chapters");

    await page.getByRole("button", { name: "Publish episode" }).click();
    await expect(page.getByRole("button", { name: "Unpublish episode" })).toBeVisible();
    await expect(page.locator(".page-header").getByText("Published", { exact: true })).toBeVisible();

    await shots.capture(
      "episode-published",
      "Episode published",
      "Publish checklist complete and the episode status changed to Published.",
    );
  });

  await test.step("unpublish and publish again", async () => {
    await page.getByRole("button", { name: "Unpublish episode" }).click();
    await expect(page.getByRole("button", { name: "Publish episode" })).toBeVisible();
    await expect(page.locator(".page-header").getByText("Draft", { exact: true })).toBeVisible();

    await shots.capture(
      "episode-unpublished",
      "Episode unpublished",
      "Episode returned to Draft with the publish action available again.",
    );

    await page.getByRole("button", { name: "Publish episode" }).click();
    await expect(page.getByRole("button", { name: "Unpublish episode" })).toBeVisible();
    await expect(page.locator(".page-header").getByText("Published", { exact: true })).toBeVisible();

    await shots.capture(
      "episode-republished",
      "Episode republished",
      "Episode published again from the editor after being unpublished.",
    );
  });

  await test.step("check the episode on the series detail page", async () => {
    await page.getByRole("link", { name: seriesTitle }).click();
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();

    const row = page.locator("tbody tr").filter({ hasText: episodeTitle });
    await expect(row).toBeVisible();
    const cells = row.locator("td");
    await expect(cells.nth(2)).toHaveText("Published");
    await expect(cells.nth(3)).toHaveText("0:20");
    await expect(cells.nth(4)).toHaveText("4");
    await expect(cells.nth(5)).toContainText("Audio uploaded");

    await shots.capture(
      "series-detail-published",
      "Published episode in series",
      "Series detail table showing the published episode with duration, chapter count and audio marker.",
    );
  });

  await test.step("edit the series", async () => {
    await page.getByRole("link", { name: "Edit series" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Edit series" })).toBeVisible();
    await expect(page.getByText(/^Version \d+$/)).toBeVisible();

    await page.getByLabel("Description", { exact: true }).fill(updatedDescription);

    await shots.capture(
      "series-edit",
      "Series edit form",
      "Edit series form with the current version and an updated description.",
    );

    await page.getByRole("button", { name: "Save changes" }).click();
    await expect(page).toHaveURL(new RegExp(`/admin/series/${seriesId}$`));
    await expect(page.getByText(updatedDescription, { exact: true })).toBeVisible();

    await shots.capture(
      "series-updated",
      "Series updated",
      "Series detail page showing the saved description.",
    );
  });

  await test.step("confirm the series cannot be deleted while it has episodes", async () => {
    await page.getByRole("link", { name: "Edit series" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Edit series" })).toBeVisible();

    await expect(page.getByRole("button", { name: "Delete series" })).toBeDisabled();
    await expect(page.getByText(/still has 1 episode/)).toBeVisible();

    await shots.capture(
      "series-delete-blocked",
      "Series deletion blocked",
      "Delete series disabled because the series still has an episode, with an explanatory hint.",
    );

    await page.getByRole("link", { name: "Cancel" }).click();
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();
  });

  await test.step("delete the episode", async () => {
    await page.getByRole("link", { name: episodeTitle }).click();
    await expect(page.getByRole("heading", { level: 1, name: episodeTitle })).toBeVisible();

    await page.getByRole("button", { name: "Delete episode" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("heading")).toContainText(episodeTitle);

    await shots.capture(
      "delete-episode-dialog",
      "Delete episode confirmation",
      "Confirmation dialog shown before deleting the episode.",
    );

    await dialog.getByRole("button", { name: "Delete episode" }).click();
    await expect(page).toHaveURL(new RegExp(`/admin/series/${seriesId}$`));
    await expect(page.getByRole("heading", { level: 1, name: seriesTitle })).toBeVisible();
    await expect(page.getByText("No episodes yet")).toBeVisible();

    await shots.capture(
      "episode-deleted",
      "Episode deleted",
      "Series detail page back to the empty episodes state after deleting the episode.",
    );
  });

  await test.step("delete the series", async () => {
    await page.getByRole("link", { name: "Edit series" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Edit series" })).toBeVisible();

    const deleteButton = page.getByRole("button", { name: "Delete series" });
    await expect(deleteButton).toBeEnabled();
    await deleteButton.click();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Delete series" }).click();

    await expect(page).toHaveURL(/\/admin\/?$/);
    await expect(page.getByRole("heading", { level: 1, name: "Series" })).toBeVisible();

    await page.getByLabel("Search series", { exact: true }).fill(seriesTitle);
    await expect(page.getByRole("heading", { name: /No series match/ })).toContainText(seriesTitle);

    await shots.capture(
      "series-deleted",
      "Series deleted",
      "Series list search showing no match after the series and its episode were deleted.",
    );
  });
});
