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

import { type Action, set, type Stored } from "@/channel/actions";
import { isCalculated } from "@/channel/payload";
import { createOf, reduceAll } from "@/channel/reduce";

const create = (stored: Partial<Stored>): Action => ({
  type: "create",
  create: {
    channel: {
      name: "created",
      leaseholder: 1,
      dataType: "float32",
      isIndex: false,
      localKey: 1,
      localIndex: 0,
      virtual: true,
      concurrency: control.Concurrency.exclusive,
      expression: "",
      ...stored,
    },
  },
});

describe("channel reduce breaker 3", () => {
  const created = createOf(create({}));
  if (created == null) throw new Error("expected a created channel");
  // A reader that a retrieve showed the output as calculated with no expression.
  const hidden = { ...created, calculated: true };

  it("should keep calculated through an operations set with no expression", () => {
    const { next } = reduceAll(hidden, [set({ path: ["operations"], value: [] })]);
    expect(isCalculated(next)).toBe(true);
  });
});
