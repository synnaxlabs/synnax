// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CDN } from "@/components/media/cdn";
import { type Check } from "@/util/checks/check";
import { locate } from "@/util/checks/crawl";
import { attrOf, attrValues } from "@/util/checks/html";

interface Ref {
  route: string;
  url: string;
  /** String to search for in the page source to resolve a line number. */
  needle: string;
}

// The page source names a docs asset by its id, not its URL.
const needle = (url: string): string =>
  url.startsWith(CDN.ROOT)
    ? url
        .slice(CDN.ROOT.length + 1)
        .replace(/#.*$/, "")
        .replace(/(\/[0-9a-f]{12})?(-light|-dark)?\.\w+$/, "")
    : url;

interface Unnamed {
  /** The source URL of the image or the first source of the video. */
  url: string;
  problem: string;
}

const unnamed = (html: string): Unnamed[] => [
  ...[...html.matchAll(/<img\b[^>]*>/g)]
    .filter(([tag]) => attrOf(tag, "alt") == null)
    .map(([tag]) => ({
      url: attrOf(tag, "src") ?? "",
      problem: "image has no alt text",
    })),
  ...[...html.matchAll(/<video\b[^>]*>/g)]
    .filter(([tag]) => attrOf(tag, "aria-label") == null)
    .map(({ index }) => {
      const source = /<source\b[^>]*>/.exec(html.slice(index))?.[0] ?? "";
      return { url: attrOf(source, "src") ?? "", problem: "video has no label" };
    }),
];

// Verifies every image, video, and poster a page references loads, and that each image
// and video has a name for screen readers.
export const media = (fullCrawl: boolean): Check => {
  const refs: Ref[] = [];
  return {
    name: "media",
    page: ({ route, html }) => {
      const urls = [
        ...["img", "video", "source"].flatMap((tag) => attrValues(html, tag, "src")),
        ...attrValues(html, "source", "srcset").flatMap((set) =>
          set.split(",").map((candidate) => candidate.trim().split(/\s+/)[0]),
        ),
        ...attrValues(html, "video", "poster"),
      ];
      // A media fragment such as #t=0.001 seeks the video; it names no anchor.
      for (const url of urls)
        refs.push({ route, url: url.replace(/#.*$/, ""), needle: needle(url) });
      return unnamed(html).map(
        ({ url, problem }) => `${locate(route, needle(url))} - ${problem}: ${url}`,
      );
    },
    finish: async (ctx, report, progress) => {
      // Canary for the media markup silently matching nothing.
      if (fullCrawl && !refs.some(({ url }) => url.startsWith(CDN.ROOT)))
        report("no docs media found: markup has changed");
      const seen = new Set<string>();
      let done = 0;
      await Promise.all(
        refs.map(async ({ route, url, needle }) => {
          try {
            const key = `${route}|${url}`;
            if (seen.has(key)) return;
            seen.add(key);
            const absolute = url.startsWith("/") ? `${ctx.baseURL}${url}` : url;
            if (!absolute.startsWith("http")) return;
            const reason = await ctx.fetchOk(absolute);
            if (reason != null) report(`${locate(route, needle)} - ${reason}`);
          } finally {
            progress(++done, refs.length);
          }
        }),
      );
    },
  };
};
