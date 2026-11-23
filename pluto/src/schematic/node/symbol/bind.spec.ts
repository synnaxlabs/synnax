// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { bindSpec, dataMissing, sources, values } from "@/schematic/node/symbol/bind";

const SPEC: schematic.symbol.Spec = {
  drawing: { type: "svg", markup: "<svg></svg>" },
  parts: [],
  handles: [],
  states: [
    {
      type: "enumeration",
      key: "active",
      name: "Active",
      enumeration: "a",
      match: { type: "exact" },
      default: 0,
    },
    { type: "number", key: "level", name: "Level", bounds: { lower: 0, upper: 100 } },
  ],
  commands: [],
  looks: [],
  interactions: [],
  variant: "static",
  scale: 1,
  strokeScaled: false,
};

describe("bindSpec", () => {
  it("should replace what a binding sets on each state", () => {
    const within = { type: "within" as const, bounds: { lower: 1, upper: 2 } };
    const bound = bindSpec(SPEC, [
      { key: "active", enumeration: "b", match: within, default: 1 },
      { key: "level", bounds: { lower: 5, upper: 10 } },
    ]);
    expect(bound.states).toEqual([
      { ...SPEC.states[0], enumeration: "b", match: within, default: 1 },
      { ...SPEC.states[1], bounds: { lower: 5, upper: 10 } },
    ]);
  });

  it("should keep a state that no binding names", () => {
    expect(bindSpec(SPEC, []).states).toEqual(SPEC.states);
  });
});

describe("sources", () => {
  it("should read only the states bound to a channel", () => {
    expect(
      Object.keys(
        sources([
          { key: "active", source: { type: "channel", key: 1 } },
          { key: "level", source: { type: "local", value: 3 } },
          { key: "other", source: { type: "channel" } },
        ]),
      ),
    ).toEqual(["active"]);
  });
});

describe("values", () => {
  it("should give a local value and the value of a channel reading", () => {
    const got = values(
      [
        { key: "active", source: { type: "channel", key: 1 } },
        { key: "level", source: { type: "local", value: 3 } },
      ],
      { active: { value: 1, stale: false } },
    );
    expect(got).toEqual(
      new Map([
        ["active", 1],
        ["level", 3],
      ]),
    );
  });

  it("should give no value for a channel with no reading or no data", () => {
    const bindings: Pick<schematic.StateBinding, "key" | "source">[] = [
      { key: "active", source: { type: "channel", key: 1 } },
      { key: "level", source: { type: "channel", key: 2 } },
    ];
    expect(values(bindings, { active: { value: null, stale: false } })).toEqual(
      new Map(),
    );
  });
});

describe("dataMissing", () => {
  const CHANNEL = { key: "active", source: { type: "channel" as const, key: 1 } };

  it.each([
    ["no reading", {}],
    ["a reading with no value", { active: { value: null, stale: false } }],
  ])("should be true for a channel with %s", (_, readings) => {
    expect(dataMissing([CHANNEL], readings)).toBe(true);
  });

  it("should be false for a channel with a value", () => {
    expect(dataMissing([CHANNEL], { active: { value: 0, stale: true } })).toBe(false);
  });

  it.each([
    ["no source", { key: "active" }],
    ["a channel source with no key", { key: "active", source: { type: "channel" } }],
    ["a local source", { key: "active", source: { type: "local", value: 1 } }],
  ] as const)("should be false for a state with %s", (_, binding) => {
    expect(dataMissing([binding], {})).toBe(false);
  });
});
