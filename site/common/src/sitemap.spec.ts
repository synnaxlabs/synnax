// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { sitemap } from "./sitemap";

const SITE = new URL("https://synnaxlabs.com");

describe("sitemap", () => {
  it("should list each route as an absolute URL on the site", async () => {
    const response = sitemap(["/", "/company"])({ site: SITE });
    expect(await response.text()).toBe(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>https://synnaxlabs.com/</loc></url>
<url><loc>https://synnaxlabs.com/company</loc></url>
</urlset>
`);
  });

  it("should respond with XML", () => {
    const response = sitemap([])({ site: SITE });
    expect(response.headers.get("Content-Type")).toBe("application/xml; charset=utf-8");
  });
});
