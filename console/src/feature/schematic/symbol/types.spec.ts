// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";

const createSymbol = (
  variant: string,
  states: schematic.symbol.State[],
): schematic.symbol.Symbol => ({
  key: "00000000-0000-0000-0000-000000000000",
  name: "sym",
  data: { svg: "<svg/>", variant, states, handles: [], scale: 1, scaleStroke: false },
});

describe("Schematic.Symbol.isStatic", () => {
  it("treats the static variant as static", () => {
    expect(
      Schematic.Symbol.isStatic(
        createSymbol("static", [
          { key: "base", name: "Base", regions: [] },
          { key: "active", name: "Active", regions: [] },
        ]),
      ),
    ).toBe(true);
  });

  it("treats a single-state actuator as static", () => {
    expect(
      Schematic.Symbol.isStatic(
        createSymbol("actuator", [{ key: "base", name: "Base", regions: [] }]),
      ),
    ).toBe(true);
  });

  it("treats a multi-state actuator as dynamic", () => {
    expect(
      Schematic.Symbol.isStatic(
        createSymbol("actuator", [
          { key: "base", name: "Base", regions: [] },
          { key: "active", name: "Active", regions: [] },
        ]),
      ),
    ).toBe(false);
  });
});
