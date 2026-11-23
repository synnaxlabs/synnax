// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type Action, set } from "@/channel/actions";
import { isCalculated } from "@/channel/payload";
import { createOf, reduceAll } from "@/channel/reduce";

const create = (expression: string): Action => ({
  type: "create",
  create: {
    channel: {
      name: "output",
      leaseholder: 1,
      dataType: "float32",
      isIndex: false,
      localKey: 7,
      localIndex: 8,
      virtual: true,
      concurrency: control.Concurrency.exclusive,
      expression,
    },
  },
});

describe("channel reduce check", () => {
  // The old API answers a subject who cannot read the calculation with calculated and
  // no expression or operations.
  const hidden = () => {
    const created = createOf(create(""));
    if (created == null) throw new Error("expected a created channel");
    return { ...created, calculated: true };
  };

  it("should keep calculated when an operations edit reaches a hidden reader", () => {
    // The edit drops the reset channel that the subject could not read, so the Core
    // sends the subject a set of the operations alone.
    const operations = set({
      path: ["operations"],
      value: [{ type: "max", reset_channel: 0, duration: "1000000000" }],
    });
    const { next } = reduceAll(hidden(), [operations]);
    expect(isCalculated(next)).toBe(true);
  });
});
