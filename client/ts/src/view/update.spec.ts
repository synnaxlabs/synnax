// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { view } from "@/view";
import { reduceAll } from "@/view/actions";

describe("reduceAll", () => {
  const createView = (): view.View =>
    view.viewZ.parse({
      key: "00000000-0000-4000-8000-000000000003",
      name: "Ranges",
      type: "range",
    });

  it("should rename a view without a type and keep its variant", () => {
    const prev = createView();
    const { next, inverse, targets } = reduceAll(prev, [
      view.update({ value: { name: "Runs" } }),
    ]);
    expect(next).toEqual({ ...prev, name: "Runs" });
    expect(targets).toEqual(["name"]);
    expect(reduceAll(next, inverse).next).toEqual(prev);
  });

  it("should change the variant of a view and keep its base fields", () => {
    const prev = createView();
    const { next } = reduceAll(prev, [view.update({ value: { type: "status" } })]);
    expect(next).toEqual({ key: prev.key, name: "Ranges", type: "status" });
  });
});
