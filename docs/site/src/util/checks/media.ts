// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { mediaURL } from "../../components/media/url.ts";
import { type Check } from "./check.ts";
import { locate } from "./crawl.ts";
import { attrValues, islands } from "./html.ts";

const THEMES = ["light", "dark"];

const islandProps = z.object({
  id: z.string(),
  themed: z.boolean().default(true),
  extension: z.string().default("png"),
});

interface Ref {
  route: string;
  url: string;
  /** String to search for in the page source to resolve a line number. */
  needle: string;
}

// Image and Video islands are client:only, so their assets never appear in server
// HTML; rebuild the CDN URLs they will request from their island props.
export const media = (fullCrawl: boolean): Check => {
  const refs: Ref[] = [];
  let matched = 0;
  return {
    name: "media",
    page: ({ route, html }) => {
      const failures: string[] = [];
      for (const island of islands(html)) {
        const kind = island.component.replace(/^.*\./, "");
        if (kind !== "Image" && kind !== "Video") continue;
        matched += 1;
        const parsed = islandProps.safeParse(island.props);
        if (!parsed.success) {
          failures.push(
            `${locate(route, `<${island.component}`)} - ` +
              `${kind} island has malformed props`,
          );
          continue;
        }
        const { id, themed } = parsed.data;
        const extension = kind === "Video" ? "mp4" : parsed.data.extension;
        const urls = themed
          ? THEMES.map((theme) => mediaURL(id, extension, theme))
          : [mediaURL(id, extension)];
        for (const url of urls) refs.push({ route, url, needle: id });
      }
      for (const tag of ["img", "video", "source"])
        for (const src of attrValues(html, tag, "src"))
          refs.push({ route, url: src, needle: src });
      for (const poster of attrValues(html, "video", "poster"))
        refs.push({ route, url: poster, needle: poster });
      return failures;
    },
    finish: async (ctx, report, progress) => {
      // Canary for the island markup silently matching nothing (see tabs.ts).
      if (fullCrawl && matched === 0)
        report("no Image or Video islands found: markup has changed");
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
