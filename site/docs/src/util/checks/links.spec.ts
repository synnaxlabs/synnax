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
  warnings: string[];
  fetched: string[];
}

// Crawls a target page holding the anchor "present" and a source page linking to href.
// Every fetch succeeds except those listed in dead.
const run = async (href: string, dead: string[] = [], strict = false): Promise<Run> => {
  const check = links(false, strict);
  check.page?.({ route: TARGET, html: '<h2 id="present">Present</h2>' });
  check.page?.({ route: SOURCE, html: `<a href="${href}">link</a>` });
  const fetched: string[] = [];
  const warnings: string[] = [];
  const ctx: Context = {
    routes: new Set([TARGET, SOURCE]),
    baseURL: BASE_URL,
    fetchOk: vi.fn(async (url: string) => {
      fetched.push(url);
      return dead.includes(url) ? "HTTP 404" : null;
    }),
    warn: (message) => warnings.push(message),
  };
  const failures: string[] = [];
  await check.finish?.(
    ctx,
    (message) => failures.push(message),
    () => {},
  );
  return { failures, warnings, fetched };
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

  describe("broken internal links", () => {
    it("should fail a dead link when not strict", async () => {
      const href = "/spec/links/missing";
      const { failures, warnings } = await run(href, [`${BASE_URL}${href}`]);
      expect(failures).toEqual([`${SOURCE} - dead internal link ${href} (HTTP 404)`]);
      expect(warnings).toEqual([]);
    });

    it("should fail a missing anchor when not strict", async () => {
      const href = `${TARGET}#absent`;
      const { failures, warnings } = await run(href);
      expect(failures).toEqual([`${SOURCE} - missing anchor ${href}`]);
      expect(warnings).toEqual([]);
    });
  });

  it.each([
    "https://synnaxlabs.com/",
    "https://www.synnaxlabs.com/company",
    "https://portal.synnaxlabs.com/login",
  ])("should not fetch %s", async (href) => {
    const { failures, fetched } = await run(href, [href]);
    expect(failures).toEqual([]);
    expect(fetched).not.toContain(href);
  });

  describe("broken external links", () => {
    const href = "https://example.com/page";

    it("should warn and not fail when not strict", async () => {
      const { failures, warnings, fetched } = await run(href, [href]);
      expect(fetched).toContain(href);
      expect(failures).toEqual([]);
      expect(warnings).toEqual([`${SOURCE} - HTTP 404`]);
    });

    it("should fail when strict", async () => {
      const { failures, warnings } = await run(href, [href], true);
      expect(failures).toEqual([`${SOURCE} - HTTP 404`]);
      expect(warnings).toEqual([]);
    });
  });
});
