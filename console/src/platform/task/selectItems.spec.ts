// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

describe("selectItems", () => {
  it("should render one item per entry in declaration order", () => {
    const items = Task.selectItems({ Volts: "V", Amps: "A" });
    expect(items.map(({ key, props }) => ({ key, props }))).toEqual([
      { key: "Volts", props: { itemKey: "Volts", children: "V" } },
      { key: "Amps", props: { itemKey: "Amps", children: "A" } },
    ]);
  });

  it("should render nothing for an empty lookup", () => {
    expect(Task.selectItems({})).toEqual([]);
  });
});
