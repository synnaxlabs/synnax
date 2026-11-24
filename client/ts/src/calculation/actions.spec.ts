// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { calculation } from "@/calculation";

const KEY = "0198b5a4-0000-7000-8000-000000000001";

describe("calculation actions", () => {
  it("should drop the status that a create action carries", () => {
    const held = calculation.calculationZ.parse({
      key: KEY,
      output: 1,
      expression: "return 1",
    });
    const created = calculation.calculationZ.parse({
      key: KEY,
      output: 1,
      expression: "return 2",
      status: {
        key: "5c1e7a0b-2d3f-4e6a-8b9c-0d1e2f3a4b5c",
        name: "Calculation status",
        variant: "info",
        message: "carried by the create",
        description: "",
        time: TimeStamp.now(),
        labels: [],
        owner: calculation.resourceID(KEY),
      },
    });
    const { next } = calculation.reduceAll(held, [
      calculation.create({ calculation: created }),
    ]);
    expect(next.expression).toEqual("return 2");
    expect(next.status).toBeUndefined();
  });
});
