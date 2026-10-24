// Post-build step for GitHub Pages: SPA fallback for deep links and .nojekyll.
import { copyFile, writeFile } from "node:fs/promises";

await copyFile("dist/index.html", "dist/404.html");
await writeFile("dist/.nojekyll", "");
