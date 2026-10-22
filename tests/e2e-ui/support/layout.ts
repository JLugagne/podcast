import type { Page } from "@playwright/test";

/** One layout defect found on the page. */
export interface LayoutIssue {
  rule: "page-overflow" | "squeezed-text" | "text-overflow";
  element: string;
  detail: string;
}

/**
 * Finds layout defects a green functional run does not catch: the page
 * scrolling sideways, text squeezed to (almost) zero width, and text spilling
 * out of its box. Elements that clip on purpose (overflow other than
 * `visible`, e.g. ellipsis) are only checked for being squeezed.
 */
export async function findLayoutIssues(page: Page): Promise<LayoutIssue[]> {
  return page.evaluate(() => {
    const issues: LayoutIssue[] = [];
    // The browser's raw XML viewer (RSS feed) is not our layout.
    if (!document.contentType.includes("html")) return issues;

    const root = document.documentElement;
    if (root.scrollWidth > window.innerWidth + 1) {
      issues.push({
        rule: "page-overflow",
        element: "html",
        detail: `page is ${root.scrollWidth}px wide for a ${window.innerWidth}px viewport`,
      });
    }

    const describe = (el: Element): string => {
      const id = el.id ? `#${el.id}` : "";
      const cls =
        typeof el.className === "string" && el.className.trim()
          ? `.${el.className.trim().split(/\s+/).join(".")}`
          : "";
      const text = (el.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 40);
      return `${el.tagName.toLowerCase()}${id}${cls} "${text}"`;
    };

    const skipped = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "TEXTAREA", "PRE", "CODE", "OPTION", "svg"]);
    for (const el of Array.from(document.body.querySelectorAll<HTMLElement>("*"))) {
      if (skipped.has(el.tagName) || el.closest("pre, textarea, svg, [aria-hidden='true']")) continue;
      const ownText = Array.from(el.childNodes)
        .filter((node) => node.nodeType === Node.TEXT_NODE)
        .map((node) => node.textContent ?? "")
        .join("")
        .trim();
      if (ownText.length < 2) continue;

      const style = getComputedStyle(el);
      if (style.display === "inline" || style.display === "none" || style.visibility === "hidden") continue;
      const rect = el.getBoundingClientRect();
      if (rect.height === 0) continue;
      // Visually hidden helpers (sr-only) are 1px boxes on purpose.
      if (rect.width <= 1 && rect.height <= 1) continue;

      if (rect.width < 16) {
        issues.push({
          rule: "squeezed-text",
          element: describe(el),
          detail: `text box is only ${Math.round(rect.width)}px wide`,
        });
        continue;
      }
      if (style.overflowX === "visible" && el.scrollWidth > el.clientWidth + 1) {
        issues.push({
          rule: "text-overflow",
          element: describe(el),
          detail: `content is ${el.scrollWidth}px wide in a ${el.clientWidth}px box`,
        });
      }
    }
    return issues;
  });
}

/** Throws with a readable list when the page has layout defects. */
export async function assertCleanLayout(page: Page, where: string): Promise<void> {
  const issues = await findLayoutIssues(page);
  if (issues.length === 0) return;
  const lines = issues.slice(0, 15).map((i) => `  - [${i.rule}] ${i.element}: ${i.detail}`);
  throw new Error(`Layout defects on "${where}" (${issues.length}):\n${lines.join("\n")}`);
}
