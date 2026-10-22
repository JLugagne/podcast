import { describe, expect, it } from "vitest";
import { buildUploadForm } from "./upload";

describe("buildUploadForm", () => {
  it("appends the text fields before the file part", () => {
    // The server streams the file part and never reads parts after it,
    // so duration_seconds must come first or it is silently dropped.
    const file = new File(["abc"], "pilot.mp3", { type: "audio/mpeg" });
    const form = buildUploadForm("file", file, { duration_seconds: "123.4" });

    expect([...form.keys()]).toEqual(["duration_seconds", "file"]);
    expect(form.get("duration_seconds")).toBe("123.4");
    expect((form.get("file") as File).name).toBe("pilot.mp3");
  });
});
