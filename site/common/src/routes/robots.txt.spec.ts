// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { GET } from "./robots.txt";

describe("robots.txt", () => {
  const response = GET({ site: new URL("https://synnaxlabs.com") });

  it("should respond with plain text", () => {
    expect(response.headers.get("Content-Type")).toBe("text/plain; charset=utf-8");
  });

  it("should allow every crawler and point at the site's sitemap", async () => {
    expect(await response.text()).toBe(`User-agent: *
Allow: /

Sitemap: https://synnaxlabs.com/sitemap.xml
`);
  });
});
