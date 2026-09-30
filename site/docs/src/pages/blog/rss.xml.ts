// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { escape } from "html-escaper";

import { POSTS } from "@/util/pages";

export const prerender = true;

export const GET: APIRoute = ({ site }) => {
  const blog = new URL("/blog", site).href;
  const items = POSTS.map(({ route, title, description, published }) => {
    const link = new URL(route, site).href;
    const date = published == null ? "" : new Date(published).toUTCString();
    return `<item>
<title>${escape(title)}</title>
<link>${link}</link>
<guid>${link}</guid>
<description>${escape(description ?? "")}</description>
<pubDate>${date}</pubDate>
</item>`;
  });
  return new Response(
    `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
<title>Synnax blog</title>
<link>${blog}</link>
<description>Articles and insights from the Synnax team.</description>
<language>en-us</language>
<atom:link href="${blog}/rss.xml" rel="self" type="application/rss+xml" />
${items.join("\n")}
</channel>
</rss>
`,
    { headers: { "Content-Type": "application/rss+xml; charset=utf-8" } },
  );
};
