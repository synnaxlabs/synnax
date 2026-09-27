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

import { aether } from "@/aether/aether";

describe("delta", () => {
  it("should return only the top-level fields that changed", () => {
    expect(aether.delta({ x: 1, y: "a" }, { x: 2, y: "a" })).toEqual({ x: 2 });
  });

  it("should return no fields when nothing changed", () => {
    expect(aether.delta({ x: 1 }, { x: 1 })).toEqual({});
  });

  it("should compare nested values deeply", () => {
    const prev = { box: { x: 1, y: 2 }, list: [1, 2] };
    const next = { box: { x: 1, y: 2 }, list: [1, 3] };
    expect(aether.delta(prev, next)).toEqual({ list: [1, 3] });
  });

  it("should compare values that define equals by value", () => {
    const prev = { at: TimeStamp.seconds(1) };
    expect(aether.delta(prev, { at: TimeStamp.seconds(1) })).toEqual({});
  });

  it("should return a field that next adds", () => {
    expect(aether.delta({ x: 1 }, { x: 1, y: "a" })).toEqual({ y: "a" });
  });

  it("should map a field that next removes to undefined", () => {
    const changed = aether.delta({ x: 1, y: "a" }, { x: 1 });
    expect(changed).toHaveProperty("y", undefined);
    expect(Object.keys(changed)).toEqual(["y"]);
  });
});
