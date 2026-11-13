// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { text } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";
import { SIZE_LEVELS } from "@/schematic/node/common/size";

describe("readoutLevel", () => {
  it.each(Component.SIZES)(
    "should keep a saved %s field at its readout level",
    (size) =>
      expect(Primitive.readoutLevel(size, Component.SIZE_TEXT_LEVELS[size])).toBe(
        text.downLevel(Component.SIZE_TEXT_LEVELS[size]),
      ),
  );

  it.each([
    ["tiny", "tiny"],
    ["small", "small"],
    ["medium", "p"],
    ["large", "h4"],
    ["huge", "h3"],
  ] as const)("should fit a standard %s field's readout at %s", (size, level) =>
    expect(Primitive.readoutLevel(size, SIZE_LEVELS[size])).toBe(level),
  );
});
