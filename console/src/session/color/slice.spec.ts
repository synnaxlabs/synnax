// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Color } from "@/session/color";

describe("Color Slice", () => {
  it("should start with no recent colors", () => {
    expect(Color.ZERO_SLICE_STATE.context.frequent).toEqual({});
  });

  // Releases through 0.57 stored a palettes record beside the colors. Nothing ever
  // wrote it, so it was dropped; the colors beside it must survive the read.
  it("should keep the recent colors stored beside a dropped palettes record", () => {
    const stored = {
      version: 0,
      context: {
        frequent: { "#FF0000": { lastUsed: 1, count: 2, relevance: 3 } },
        palettes: { frequent: { key: "frequent", name: "Frequent", swatches: [] } },
      },
    };
    expect(Color.sliceStateZ.parse(stored).context).toEqual({
      frequent: { "#FF0000": { lastUsed: 1, count: 2, relevance: 3 } },
    });
  });
});
