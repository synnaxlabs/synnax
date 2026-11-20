// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { indexName, parse, resolve, ROOT, rootOf, withSites } from "@/util/version";

const TRAINS = ["0.61", "0.60", "0.59"];

describe("version", () => {
  describe("parse", () => {
    it("should return null for an unversioned build", () => {
      expect(parse(undefined)).toBeNull();
      expect(parse("")).toBeNull();
    });

    it("should accept a train and next", () => {
      expect(parse("0.61")).toEqual("0.61");
      expect(parse("next")).toEqual("next");
    });

    it("should reject anything else", () => {
      expect(() => parse("0.61.2")).toThrow(
        "DOCS_VERSION must be a train such as 0.61 or next, not 0.61.2",
      );
    });
  });

  describe("resolve", () => {
    it("should serve the newest train from the root", () => {
      expect(resolve("0.61", TRAINS, "0.61.0")).toEqual({
        name: "0.61",
        url: ROOT,
        status: "latest",
        release: "0.61.0",
        versions: [
          { name: "next", url: "https://next.docs.synnaxlabs.com" },
          { name: "0.61", url: ROOT },
          { name: "0.60", url: "https://v0-60.docs.synnaxlabs.com" },
        ],
      });
    });

    it("should mark an earlier train as old", () => {
      const version = resolve("0.60", TRAINS, "0.60.0");
      expect(version.status).toEqual("old");
      expect(version.url).toEqual("https://v0-60.docs.synnaxlabs.com");
    });

    it("should mark next", () => {
      expect(resolve("next", TRAINS, null).status).toEqual("next");
    });

    it("should list an old train that left the menu while its own site shows", () => {
      expect(
        resolve("0.59", TRAINS, "0.59.0").versions.map(({ name }) => name),
      ).toEqual(["next", "0.61", "0.60", "0.59"]);
    });

    it("should order trains by number, not by text", () => {
      const names = resolve("0.10", ["0.10", "0.9"], "0.10.0").versions.map(
        ({ name }) => name,
      );
      expect(names).toEqual(["next", "0.10", "0.9"]);
    });

    it("should reject a train with no site", () => {
      expect(() => resolve("0.62", TRAINS, "0.62.0")).toThrow(
        "Train 0.62 has no docs site",
      );
    });

    it("should reject a build when no train has a site", () => {
      expect(() => resolve("next", [], null)).toThrow("No train has a docs site");
    });
  });

  describe("withSites", () => {
    it("should drop the trains that predate versioned docs", () => {
      expect(withSites(["1.0", "0.61", "0.60", "0.59", "0.9"])).toEqual([
        "1.0",
        "0.61",
        "0.60",
      ]);
    });
  });

  describe("rootOf", () => {
    it("should keep links on the newest train and on unversioned builds", () => {
      expect(rootOf(null)).toEqual("");
      expect(rootOf(resolve("0.61", TRAINS, "0.61.0"))).toEqual("");
    });

    it("should send other versions to the root", () => {
      expect(rootOf(resolve("0.60", TRAINS, "0.60.0"))).toEqual(ROOT);
      expect(rootOf(resolve("next", TRAINS, null))).toEqual(ROOT);
    });
  });

  describe("indexName", () => {
    it("should give each version its own index", () => {
      expect(indexName(null)).toEqual("docs_site_next");
      expect(indexName("0.61")).toEqual("docs_site_0_61");
      expect(indexName("next")).toEqual("docs_site_next");
    });
  });
});
