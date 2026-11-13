// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { manifest, version } from "@/pages/releases/_manifest";
import { Releases } from "@/util/releases";

const listing = (...tags: string[]): Releases =>
  new Releases({
    fetch: async () =>
      Response.json(
        tags.map((tag_name) => ({
          tag_name,
          draft: false,
          prerelease: false,
          published_at: null,
        })),
      ),
  });

const failing = new Releases({
  fetch: async () => new Response(null, { status: 500 }),
});

describe("releases manifest", () => {
  describe("version", () => {
    it("should answer the highest stable version to any origin", async () => {
      const res = await version(
        listing("core/v0.61.0", "core/v0.62.0-rc.1", "console/v0.63.0"),
        "core",
        "stable",
      );
      expect(res.status).toEqual(200);
      expect(await res.json()).toEqual({ version: "0.61.0" });
      expect(res.headers.get("Access-Control-Allow-Origin")).toEqual("*");
    });

    it("should answer 503 when the lookup fails", async () => {
      const res = await version(failing, "core", "stable");
      expect(res.status).toEqual(503);
      expect(await res.text()).toEqual("GitHub releases API returned 500");
    });
  });

  describe("manifest", () => {
    it("should redirect to the manifest of the highest release", async () => {
      const res = await manifest(listing("console/v0.61.0"), "console", "stable");
      expect(res.status).toEqual(302);
      expect(res.headers.get("Location")).toEqual(
        "https://github.com/synnaxlabs/synnax/releases/download/console/v0.61.0/latest.json",
      );
    });
  });
});
