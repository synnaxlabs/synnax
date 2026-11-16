// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Color } from "@/color";

describe("recalculate", () => {
  it("should rank a color used now above one used as often a year ago", () => {
    const now = Number(TimeStamp.now().valueOf());
    const yearAgo = now - Number(TimeSpan.days(365).valueOf());
    const next = Color.recalculate(10, {
      "#000000": { lastUsed: yearAgo, count: 1, relevance: 0 },
      "#ffffff": { lastUsed: now, count: 1, relevance: 0 },
    });
    // Recency carries 0.6 of the weight, so a year of disuse must cost most of it.
    expect(next["#ffffff"].relevance - next["#000000"].relevance).toBeGreaterThan(0.3);
  });

  it("should keep a color just picked over stale colors used twice", () => {
    const now = Number(TimeStamp.now().valueOf());
    const yearAgo = now - Number(TimeSpan.days(365).valueOf());
    const frequent: Color.Frequent = {};
    for (let i = 0; i < 10; i++)
      frequent[`#00000${i}`] = { lastUsed: yearAgo, count: 2, relevance: 0 };
    frequent["#ffffff"] = { lastUsed: now, count: 1, relevance: 0 };
    expect(Color.recalculate(10, frequent)).toHaveProperty("#ffffff");
  });

  it("should cap the recency of a color last used in the future at 1", () => {
    const now = Number(TimeStamp.now().valueOf());
    const day = Number(TimeSpan.days(1).valueOf());
    const next = Color.recalculate(10, {
      "#000000": { lastUsed: now - day, count: 1, relevance: 0 },
      "#ffffff": { lastUsed: now + day, count: 1, relevance: 0 },
    });
    expect(next["#ffffff"].relevance).toBeLessThanOrEqual(1);
  });
});
