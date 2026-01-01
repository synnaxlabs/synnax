// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { telem } from "@/telem/aether";

describe("telem", () => {
  it("pipeline", async () => {
    const s1 = telem.fixedNumber(20);
    const s2 = telem.fixedNumber(9);
    const avg = telem.mean({});
    const bool = telem.withinBounds({ trueBound: { upper: 15, lower: 5 } });
    const p = new telem.SourcePipeline(
      {
        connections: [
          { from: "s1", to: "avg" },
          { from: "s2", to: "avg" },
          { from: "avg", to: "bool" },
        ],
        outlet: "bool",
        segments: { s1, s2, avg, bool },
      },
      telem.createFactory(),
    );
    expect(await p.value()).toBe(true);
  });
});
