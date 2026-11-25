// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/general/offPageReference/primitive";
const PAGE: schematic.Page = { type: "schematic", key: "page-key" };

describe("Primitive.offPageReferenceTooltip", () => {
  it("should return double-click tooltip by default", () => {
    expect(Primitive.offPageReferenceTooltip(PAGE)).toBe("Double-click to navigate");
  });

  it("should return single-click tooltip when double-click nav is disabled", () => {
    expect(Primitive.offPageReferenceTooltip(PAGE, true)).toBe(
      "Single-click to navigate",
    );
  });

  it("should return double-click tooltip when double-click nav is enabled", () => {
    expect(Primitive.offPageReferenceTooltip(PAGE, false)).toBe(
      "Double-click to navigate",
    );
  });

  it("should return undefined when the page key is empty", () => {
    expect(
      Primitive.offPageReferenceTooltip({ type: "schematic", key: "" }),
    ).toBeUndefined();
  });
});
