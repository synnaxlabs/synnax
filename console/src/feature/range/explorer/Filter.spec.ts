// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { PROPERTIES } from "@/feature/range/explorer/Filter";

const A = "4a1f0f4e-2b8f-4d3c-9b8e-3f6c2a1d0e01";

describe("range/explorer/Filter", () => {
  it.each<[string, unknown]>([
    ["stage", ["in_progress", "completed"]],
    ["time", TimeSpan.days(7)],
    ["time", new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2))],
    ["parent", [A]],
  ])("should read back a %s condition", (property, value) => {
    const { encode, decode } = PROPERTIES[property];
    expect(decode(encode(value))).toEqual(value);
  });

  it.each<[string, ranger.Where]>([
    ["stage", { stages: ["to_do"], parents: [A] }],
    ["time", { labels: [A] }],
    ["parent", { stages: ["to_do"] }],
  ])("should leave a node that is not a %s condition to another chip", (p, node) => {
    expect(PROPERTIES[p].decode(node)).toBeNull();
  });
});
