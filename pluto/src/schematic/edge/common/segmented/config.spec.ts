// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { createDefaultConfig } from "@/schematic/edge/common/segmented/config";

describe("createDefaultConfig", () => {
  it("should leave the color absent so the theme picks it", () => {
    const config = createDefaultConfig("pipe");
    expect(config.variant).toBe("pipe");
    expect(config.segments).toEqual([]);
    expect(config.color).toBeUndefined();
  });
});
