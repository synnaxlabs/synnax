// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { symbolColorVar } from "@/schematic/symbolColor";

describe("symbolColorVar", () => {
  it("should carry the rgb channels and an opaque alpha", () => {
    expect(symbolColorVar("#ff0000")).toBe("255, 0, 0, 1");
  });

  it("should carry a translucent alpha", () => {
    expect(symbolColorVar([255, 0, 0, 0.5])).toBe("255, 0, 0, 0.5");
  });

  it("should be unset for the ZERO sentinel", () => {
    expect(symbolColorVar(color.ZERO)).toBeUndefined();
  });

  it("should be unset for no color", () => {
    expect(symbolColorVar(undefined)).toBeUndefined();
  });
});
