import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import type { CDPSession, Page } from "@playwright/test";
import { assertCleanLayout } from "./layout";

/** Screenshot folder of one spec; each spec owns and cleans its own folder. */
export type ScreenshotArea = "admin" | "listener";

export interface ScreenshotEntry {
  step: number;
  file: string;
  title: string;
  caption: string;
}

export interface ScreenshotManifest {
  run: string;
  status: "passed" | "failed";
  shots: ScreenshotEntry[];
}

/** Directory receiving the numbered screenshots and manifest.json of an area. */
export function screenshotsDir(area: ScreenshotArea): string {
  return path.resolve(__dirname, "..", "screenshots", area);
}

/** Deletes and recreates the screenshot directory of one area. */
export function resetScreenshots(area: ScreenshotArea): void {
  rmSync(screenshotsDir(area), { recursive: true, force: true });
  mkdirSync(screenshotsDir(area), { recursive: true });
}

/**
 * Captures the numbered, full-page screenshots of one spec run and writes the
 * manifest consumed by the progress tracker.
 *
 * The capture goes through the Chrome DevTools Protocol: the bundled Chromium
 * build returns stale compositor frames through `page.screenshot()` in this
 * environment (the first capture of a page can show an unrelated frame).
 */
export class ScreenshotRecorder {
  readonly shots: ScreenshotEntry[] = [];
  failed = false;
  private readonly run = new Date().toISOString();
  private readonly dir: string;
  private session: CDPSession | undefined;

  constructor(
    private readonly page: Page,
    private readonly area: ScreenshotArea,
    private readonly deviceScaleFactor = 1,
  ) {
    this.dir = screenshotsDir(area);
    mkdirSync(this.dir, { recursive: true });
  }

  /** Captures one full-page screenshot and records its manifest entry. */
  async capture(slug: string, title: string, caption: string): Promise<void> {
    const step = this.shots.length + 1;
    const file = `${String(step).padStart(2, "0")}-${slug}.png`;

    // Let pending renders and repaints settle so the capture is not a stale frame.
    await this.page.evaluate(() => window.scrollTo(0, 0));
    await this.page.evaluate(
      () =>
        new Promise<void>((resolve) => {
          requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
        }),
    );
    await this.page.waitForTimeout(120);

    // Growing the viewport to the document height, then taking a viewport
    // screenshot, captures the whole page without touching the emulated
    // device metrics (full-page capture would reset the device scale factor).
    const viewport = this.page.viewportSize();
    if (!viewport) throw new Error("The page has no viewport size.");
    const height = await this.page.evaluate(() =>
      Math.max(
        document.documentElement.scrollHeight,
        document.body.scrollHeight,
        window.innerHeight,
      ),
    );
    await this.page.setViewportSize({ width: viewport.width, height });
    try {
      this.session ??= await this.page.context().newCDPSession(this.page);
      const shot = await this.session.send("Page.captureScreenshot", {
        format: "png",
        clip: { x: 0, y: 0, width: viewport.width, height, scale: this.deviceScaleFactor },
      });
      writeFileSync(path.join(this.dir, file), Buffer.from(shot.data, "base64"));
    } finally {
      await this.page.setViewportSize(viewport);
    }

    this.shots.push({ step, file, title, caption });

    // Checked after the capture so a failing step still leaves its screenshot.
    await assertCleanLayout(this.page, `${file} — ${title}`);
  }

  /** Writes <area>/manifest.json with the final status of the run. */
  writeManifest(): void {
    mkdirSync(this.dir, { recursive: true });
    const manifest: ScreenshotManifest = {
      run: this.run,
      status: this.failed ? "failed" : "passed",
      shots: this.shots,
    };
    writeFileSync(path.join(this.dir, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
  }
}
