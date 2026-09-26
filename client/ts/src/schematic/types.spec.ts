// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "zod/compile";

import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { schematic } from "@/schematic";

// Reads a Zod internal on purpose: a Zod upgrade that stops compiling variants on
// their own should fail here, not silently slow the first parse.
const isCompiled = (schema: z.ZodType): boolean =>
  (schema._zod.bag as { validator?: unknown }).validator != null;

describe("elementConfigZ", () => {
  it("Should compile only the variant a parent schema parses", () => {
    schematic.setNode({
      node: { key: "meter", position: { x: 0, y: 0 } },
      config: { variant: "flowmeter_coriolis" },
    });
    expect(isCompiled(schematic.flowmeterCoriolisElementConfigZ)).toBe(true);
    expect(isCompiled(schematic.flowmeterTurbineElementConfigZ)).toBe(false);
  });
});
