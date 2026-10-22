import { ensureFixtures } from "./fixtures/generate";

/** Regenerates the binary fixtures. Each spec cleans its own screenshot folder. */
export default async function globalSetup(): Promise<void> {
  ensureFixtures();
}
