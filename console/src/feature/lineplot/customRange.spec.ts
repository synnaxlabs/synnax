// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { type telem } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { fromWindow, toWindow } from "@/feature/lineplot/customRange";

const PAIRS: [string, lineplot.CustomRange, telem.Window][] = [
  [
    "a rolling window",
    { variant: "dynamic", span: 60 },
    { variant: "dynamic", span: 60 },
  ],
  [
    "a fixed window",
    { variant: "static", start: "2", end: "3" },
    { variant: "static", timeRange: { start: "2", end: "3" } },
  ],
];

describe("lineplot/customRange", () => {
  it.each(PAIRS)("should convert %s to a window", (_, custom, window) => {
    expect(toWindow(custom)).toEqual(window);
  });

  it.each(PAIRS)("should convert %s back from a window", (_, custom, window) => {
    expect(fromWindow(window)).toEqual(custom);
  });
});
