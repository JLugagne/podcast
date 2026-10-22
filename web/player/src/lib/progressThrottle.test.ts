import { describe, expect, it } from "vitest";
import { PROGRESS_SAVE_INTERVAL_MS, ProgressThrottle } from "./progressThrottle";

describe("ProgressThrottle", () => {
  it("allows a save once the interval has elapsed", () => {
    const throttle = new ProgressThrottle(10_000, 1_000);
    expect(throttle.shouldSave(1_000)).toBe(false);
    expect(throttle.shouldSave(10_999)).toBe(false);
    expect(throttle.shouldSave(11_000)).toBe(true);

    throttle.markSaved(11_000);
    expect(throttle.shouldSave(20_999)).toBe(false);
    expect(throttle.shouldSave(21_000)).toBe(true);
  });

  it("resets the window", () => {
    const throttle = new ProgressThrottle(5_000, 100);
    throttle.markSaved(4_000);
    expect(throttle.shouldSave(8_000)).toBe(false);
    throttle.reset(8_000);
    expect(throttle.shouldSave(12_999)).toBe(false);
    expect(throttle.shouldSave(13_000)).toBe(true);
  });

  it("defaults to the ten second interval", () => {
    expect(PROGRESS_SAVE_INTERVAL_MS).toBe(10_000);
    const throttle = new ProgressThrottle();
    expect(throttle.shouldSave(9_999)).toBe(false);
    expect(throttle.shouldSave(10_000)).toBe(true);
  });
});
