// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { formatRelativeTime } from "@/vis/axis/relative";

const MS = 1e6;
const S = 1e9;
const MIN = 60 * S;

describe("formatRelativeTime", () => {
  it("should show T0 for a value within half a step of zero", () => {
    expect(formatRelativeTime(0.4 * MS, MS)).toEqual("T0");
  });
  it("should show a positive span with T+", () => {
    expect(formatRelativeTime(12.5 * S, 0.5 * S)).toEqual("T+12.5 s");
  });
  it("should show a negative span with T-", () => {
    expect(formatRelativeTime(-250 * MS, 50 * MS)).toEqual("T-250 ms");
  });
  it("should show a span of a minute or more as a clock", () => {
    expect(formatRelativeTime(90 * S, S)).toEqual("T+1:30");
  });
  it("should show hours when the magnitude is an hour or more", () => {
    expect(formatRelativeTime(3750 * S, S)).toEqual("T+1:02:30");
  });
  it("should take its unit from the magnitude", () => {
    expect(formatRelativeTime(500 * MS, 500 * MS, 2 * S)).toEqual("T+0.5 s");
  });
  it("should show sub second precision on a clock", () => {
    expect(formatRelativeTime(MIN + 1.5 * S, 0.5 * S)).toEqual("T+1:01.5");
  });
});
