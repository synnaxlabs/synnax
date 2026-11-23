// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { docFiles, enumerateRoutes } from "@/util/checks/crawl";

describe("crawl", () => {
  describe("docFiles", () => {
    it("should list the docs pages relative to src/docs", () => {
      expect(docFiles()).toEqual(
        expect.arrayContaining([
          "reference/channels/create.mdx",
          "blog/introducing-arc/index.mdx",
        ]),
      );
    });

    it("should only list routed MDX files", () => {
      for (const f of docFiles()) {
        expect(f).toMatch(/\.mdx$/);
        expect(f).not.toMatch(/(^|\/)_/);
      }
    });
  });

  describe("enumerateRoutes", () => {
    it("should hold the routes of docs pages and Astro pages", () => {
      expect(enumerateRoutes()).toEqual(
        expect.arrayContaining([
          "/reference/channels/create",
          "/blog/introducing-arc/",
          "/privacy-policy",
        ]),
      );
    });

    it("should leave out dynamic routes, partials, and endpoints", () => {
      const routes = enumerateRoutes();
      expect(routes.filter((r) => r.includes("["))).toEqual([]);
      expect(routes.filter((r) => /(^|\/)_/.test(r))).toEqual([]);
      expect(routes).not.toContain("/llms.txt");
      expect(routes).not.toContain("/sitemap.xml");
    });

    it("should hold each route once", () => {
      const routes = enumerateRoutes();
      expect(new Set(routes).size).toBe(routes.length);
    });
  });
});
