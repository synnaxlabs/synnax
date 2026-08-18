// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { cdnKey, define, filter, videoName } from "@/manifest/manifest";

describe("manifest.define", () => {
  it("should accept well-formed entries", () => {
    const m = define([
      { id: "console/line-plots/data-tab", script: "scripts/line-plot-data.ts" },
      { id: "console/ranges/create", script: "scripts/ranges-create.ts" },
    ]);
    expect(m).toHaveLength(2);
  });

  it("should reject duplicate ids", () => {
    expect(() =>
      define([
        { id: "console/ranges/create", script: "a.ts" },
        { id: "console/ranges/create", script: "b.ts" },
      ]),
    ).toThrow("duplicate manifest id");
  });

  it("should reject ids that are not slash-separated kebab-case", () => {
    expect(() => define([{ id: "single-segment", script: "a.ts" }])).toThrow(
      "kebab-case",
    );
    expect(() => define([{ id: "console/Line Plots/x", script: "a.ts" }])).toThrow(
      "kebab-case",
    );
  });
});

describe("manifest.filter", () => {
  const m = define([
    { id: "console/ranges/create", script: "a.ts" },
    { id: "console/ranges/add-label", script: "b.ts" },
    { id: "console/users/register", script: "c.ts" },
  ]);

  it("should match by substring", () => {
    expect(filter(m, "ranges")).toHaveLength(2);
    expect(filter(m, "console/users")).toHaveLength(1);
  });

  it("should return everything without a pattern", () => {
    expect(filter(m)).toHaveLength(3);
  });
});

describe("manifest names", () => {
  it("should derive video file names and CDN keys from the id", () => {
    expect(videoName("console/ranges/create", "light")).toEqual(
      "console/ranges/create-light.mp4",
    );
    expect(cdnKey("console/ranges/create", "dark")).toEqual(
      "docs/console/ranges/create-dark.mp4",
    );
  });
});
