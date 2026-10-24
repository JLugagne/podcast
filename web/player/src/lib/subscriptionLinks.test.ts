import { describe, expect, it } from "vitest";
import { feedUrlsFromSearch, subscriptionLink } from "./subscriptionLinks";

const ONE = "https://a.example/one.xml";
const TWO = "https://b.example/two.xml";

describe("feedUrlsFromSearch", () => {
  it("reads repeated feed parameters in order", () => {
    expect(
      feedUrlsFromSearch(
        "?feed=https%3A%2F%2Fa.example%2Fone.xml&feed=https%3A%2F%2Fb.example%2Ftwo.xml",
      ),
    ).toEqual([ONE, TWO]);
  });

  it("reads a single feeds parameter split on commas and whitespace", () => {
    expect(
      feedUrlsFromSearch(
        "?feeds=https%3A%2F%2Fa.example%2Fone.xml%2C%20https%3A%2F%2Fb.example%2Ftwo.xml",
      ),
    ).toEqual([ONE, TWO]);
  });

  it("ignores blanks and duplicates", () => {
    expect(
      feedUrlsFromSearch(`?feed=&feed=${encodeURIComponent(ONE)}&feed=${encodeURIComponent(ONE)}`),
    ).toEqual([ONE]);
  });

  it("returns nothing without feed parameters", () => {
    expect(feedUrlsFromSearch("?utm_source=share")).toEqual([]);
  });
});

describe("subscriptionLink", () => {
  it("builds a link that round trips through feedUrlsFromSearch", () => {
    const link = subscriptionLink([ONE, TWO], "https://player.example/");

    expect(link).toBe(
      "https://player.example/?feed=https%3A%2F%2Fa.example%2Fone.xml&feed=https%3A%2F%2Fb.example%2Ftwo.xml",
    );
    expect(feedUrlsFromSearch(new URL(link).search)).toEqual([ONE, TWO]);
  });

  it("drops any query or hash from the base URL", () => {
    expect(subscriptionLink([ONE], "https://player.example/?x=1#top")).toBe(
      "https://player.example/?feed=https%3A%2F%2Fa.example%2Fone.xml",
    );
  });
});
