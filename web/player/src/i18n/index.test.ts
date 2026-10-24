import { afterEach, describe, expect, it } from "vitest";
import { localeFromLanguages, setLocale, t } from "./index";

afterEach(() => setLocale("en"));

describe("localeFromLanguages", () => {
  it("picks French from a French browser", () => {
    expect(localeFromLanguages(["fr-FR", "en-US"])).toBe("fr");
    expect(localeFromLanguages(["fr"])).toBe("fr");
  });

  it("uses the first supported language", () => {
    expect(localeFromLanguages(["de-DE", "en-GB", "fr-FR"])).toBe("en");
  });

  it("falls back to English", () => {
    expect(localeFromLanguages(["de-DE", "es-ES"])).toBe("en");
    expect(localeFromLanguages([])).toBe("en");
  });
});

describe("t", () => {
  it("interpolates parameters", () => {
    expect(t("chapterLine", { number: 2, title: "Tides" })).toBe("Chapter 2: Tides");
  });

  it("uses English plurals", () => {
    expect(t("episodes", { count: 1 })).toBe("1 episode");
    expect(t("episodes", { count: 2 })).toBe("2 episodes");
  });

  it("uses French plurals, singular for zero and one", () => {
    setLocale("fr");
    expect(t("episodes", { count: 0 })).toBe("0 épisode");
    expect(t("episodes", { count: 1 })).toBe("1 épisode");
    expect(t("episodes", { count: 2 })).toBe("2 épisodes");
  });

  it("translates the interface strings", () => {
    setLocale("fr");
    expect(t("listen")).toBe("Écouter");
    expect(t("addPodcast")).toBe("Ajouter un podcast");
  });
});
