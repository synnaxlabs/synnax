// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext, type APIRoute } from "astro";

/** Returns a route that lists the given paths, relative to the site URL, as a sitemap. */
export const sitemap = (routes: string[]) =>
  (({ site }: Pick<APIContext, "site">) => {
    const urls = routes.map(
      (route) => `<url><loc>${new URL(route, site).href}</loc></url>`,
    );
    return new Response(
      `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.join("\n")}
</urlset>
`,
      { headers: { "Content-Type": "application/xml; charset=utf-8" } },
    );
  }) satisfies APIRoute;
