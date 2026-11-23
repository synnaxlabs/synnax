// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { describe, expect, it } from "vitest";

import { AIN0, BLOCKS, GND } from "@/components/blog/labjack/t7";

describe("t7", () => {
  it("should wire the thermocouple to AIN0 and the GND beside it", () => {
    expect(AIN0.label).toBe("AIN0");
    expect(GND.label).toBe("GND");
    const neighbors = BLOCKS.flatMap(({ terminals }) =>
      terminals.slice(1).map((terminal, i) => [terminals[i], terminal]),
    );
    expect(neighbors).toContainEqual([AIN0, GND]);
  });
});
