// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { type Context } from "@/util/checks/check";
import { links } from "@/util/checks/links";

// No page source exists for these routes, so failures carry the bare route.
const TARGET = "/spec/links/target";
const SOURCE = "/spec/links/source";
const BASE_URL = "http://localhost:4399";

interface Run {
  failures: string[];
  fetched: string[];
}

// Crawls a target page holding the anchor "present" and a source page linking to href.
// Every fetch succeeds except those listed in dead.
const run = async (href: string, dead: string[] = []): Promise<Run> => {
  const check = links(false);
  check.page?.({ route: TARGET, html: '<h2 id="present">Present</h2>' });
  check.page?.({ route: SOURCE, html: `<a href="${href}">link</a>` });
  const fetched: string[] = [];
  const ctx: Context = {
    routes: new Set([TARGET, SOURCE]),
    baseURL: BASE_URL,
    fetchOk: vi.fn(async (url: string) => {
      fetched.push(url);
      return dead.includes(url) ? "HTTP 404" : null;
    }),
  };
  const failures: string[] = [];
  await check.finish?.(
    ctx,
    (message) => failures.push(message),
    () => {},
  );
  return { failures, fetched };
};

describe("links", () => {
  describe("docs.synnaxlabs.com links", () => {
    it("should check a link to a built page against the build", async () => {
      const href = `https://docs.synnaxlabs.com${TARGET}#present`;
      const { failures, fetched } = await run(href);
      expect(failures).toEqual([]);
      expect(
        fetched.filter((url) => new URL(url).host === "docs.synnaxlabs.com"),
      ).toEqual([]);
    });

    it("should report a missing anchor on a built page", async () => {
      const href = `https://docs.synnaxlabs.com${TARGET}#absent`;
      const { failures } = await run(href);
      expect(failures).toEqual([`${SOURCE} - missing anchor ${href}`]);
    });

    it("should fetch a route the build lacks from the local server", async () => {
      const href = "https://docs.synnaxlabs.com/spec/links/missing";
      const { failures, fetched } = await run(href, [`${BASE_URL}/spec/links/missing`]);
      expect(fetched).toContain(`${BASE_URL}/spec/links/missing`);
      expect(failures).toEqual([`${SOURCE} - dead internal link ${href} (HTTP 404)`]);
    });
  });

  it("should fetch a link to another host as an external URL", async () => {
    const href = "https://example.com/page";
    const { failures, fetched } = await run(href, [href]);
    expect(fetched).toContain(href);
    expect(failures).toEqual([`${SOURCE} - HTTP 404`]);
  });
});
