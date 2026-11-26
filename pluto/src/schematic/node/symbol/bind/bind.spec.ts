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

import { Telem } from "@/schematic/node/common/telem";
import { bind } from "@/schematic/node/symbol/bind";

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

describe("bind.spec", () => {
  it("should replace what a binding sets on each state", () => {
    const within = { type: "within" as const, bounds: { lower: 1, upper: 2 } };
    const bound = bind.spec(SPEC, [
      { key: "active", enumeration: "b", match: within, default: 1 },
      { key: "level", bounds: { lower: 5, upper: 10 } },
    ]);
    expect(bound.states).toEqual([
      { ...SPEC.states[0], enumeration: "b", match: within, default: 1 },
      { ...SPEC.states[1], bounds: { lower: 5, upper: 10 } },
    ]);
  });

  it("should keep a state that no binding names", () => {
    expect(bind.spec(SPEC, []).states).toEqual(SPEC.states);
  });
});

describe("bind.channels", () => {
  it("should list each channel that a state reads or commands once", () => {
    expect(
      bind.channels([
        { key: "active", source: { type: "channel", key: 1 } },
        { key: "level", source: { type: "local", value: 3 } },
        { key: "other", source: { type: "channel", command: 3 } },
        { key: "again", source: { type: "channel", key: 1, command: 2 } },
      ]),
    ).toEqual([1, 3, 2]);
  });
});

describe("bind.target", () => {
  const STATES = [
    { key: "active", source: { type: "channel" as const, key: 1, command: 2 } },
    { key: "level", source: { type: "local" as const, value: 3 } },
    { key: "read", source: { type: "channel" as const, key: 4 } },
  ];
  const COMMANDS = [{ key: "vent", channel: 5 }];

  it.each([
    ["the command channel of its state", { key: "open", state: "active" }, 2],
    ["the channel that binds a command with no state", { key: "vent" }, 5],
  ])("should write %s", (_, command, channel) => {
    expect(bind.target(command, STATES, COMMANDS)).toEqual({
      type: "channel",
      channel,
    });
  });

  it("should write the local value of its state", () => {
    expect(bind.target({ key: "set", state: "level" }, STATES, COMMANDS)).toEqual({
      type: "local",
      state: "level",
    });
  });

  it.each([
    ["a state with no command channel", { key: "open", state: "read" }],
    ["a state that the node does not bind", { key: "open", state: "other" }],
    ["no state and no binding", { key: "purge" }],
  ])("should give no target for %s", (_, command) => {
    expect(bind.target(command, STATES, COMMANDS)).toBeUndefined();
  });
});

describe("bind.commandChannels", () => {
  it("should give each channel that a command writes, once", () => {
    const commands: schematic.symbol.Command[] = [
      {
        key: "close",
        name: "Close",
        state: "active",
        value: { type: "fixed", value: 0 },
      },
      {
        key: "open",
        name: "Open",
        state: "active",
        value: { type: "fixed", value: 1 },
      },
      { key: "set", name: "Set", state: "level", value: { type: "ask" } },
      { key: "vent", name: "Vent", value: { type: "fixed", value: 1 } },
    ];
    expect(
      bind.commandChannels(
        { commands },
        [
          { key: "active", source: { type: "channel", key: 1, command: 2 } },
          { key: "level", source: { type: "local", value: 3 } },
        ],
        [{ key: "vent", channel: 5 }],
      ),
    ).toEqual([2, 5]);
  });
});

describe("bind.sinks", () => {
  it("should build a number sink for each channel, by its key", () => {
    expect(bind.sinks([2, 5])).toEqual({
      2: Telem.numberSink(2),
      5: Telem.numberSink(5),
    });
  });
});

describe("bind.sources", () => {
  it("should build a number source for each channel, by its key", () => {
    expect(bind.sources([1, 2])).toEqual({
      1: Telem.numberSource(1),
      2: Telem.numberSource(2),
    });
  });
});

describe("bind.values", () => {
  it("should give a local value and the reading of a state's channel", () => {
    const got = bind.values(
      [
        { key: "active", source: { type: "channel", key: 1 } },
        { key: "level", source: { type: "local", value: 3 } },
      ],
      { 1: { value: 1, stale: false } },
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
    expect(bind.values(bindings, { 1: { value: null, stale: false } })).toEqual(
      new Map(),
    );
  });
});

describe("bind.stateReadings", () => {
  it("should read a local value, a channel, and NaN for a channel with no data", () => {
    expect(
      bind.stateReadings(
        [
          { key: "active", source: { type: "channel", key: 1 } },
          { key: "level", source: { type: "local", value: 3 } },
          { key: "flow", source: { type: "channel", key: 2 } },
          { key: "speed", source: { type: "channel", key: 3 } },
        ],
        { 1: { value: 1, stale: true }, 2: { value: null, stale: false } },
      ),
    ).toEqual({
      active: { value: 1, stale: true },
      level: { value: 3 },
      flow: { value: NaN, stale: false },
      speed: { value: NaN },
    });
  });

  it("should give no reading for a state with no source", () => {
    expect(
      bind.stateReadings(
        [{ key: "active" }, { key: "level", source: { type: "channel" } }],
        {},
      ),
    ).toEqual({});
  });
});

describe("bind.stale", () => {
  const bindings: Pick<schematic.StateBinding, "key" | "source">[] = [
    { key: "active", source: { type: "channel", key: 1 } },
    { key: "level", source: { type: "local", value: 3 } },
  ];

  it("should be true while a state's channel reads stale", () => {
    expect(bind.stale(bindings, { 1: { value: 1, stale: true } })).toBe(true);
  });

  it("should be false for a stale channel that no state reads", () => {
    expect(
      bind.stale(bindings, {
        1: { value: 1, stale: false },
        9: { value: 1, stale: true },
      }),
    ).toBe(false);
  });
});

describe("bind.missing", () => {
  const CHANNEL = { key: "active", source: { type: "channel" as const, key: 1 } };

  it.each([
    ["no reading", {}],
    ["a reading with no value", { 1: { value: null, stale: false } }],
  ])("should hold a state whose channel has %s", (_, readings) => {
    expect(bind.missing([CHANNEL], readings)).toEqual(new Set(["active"]));
  });

  it("should not hold a state whose channel has a value", () => {
    expect(bind.missing([CHANNEL], { 1: { value: 0, stale: true } })).toEqual(
      new Set(),
    );
  });

  it.each([
    ["no source", { key: "active" }],
    ["a channel source with no key", { key: "active", source: { type: "channel" } }],
    ["a local source", { key: "active", source: { type: "local", value: 1 } }],
  ] as const)("should not hold a state with %s", (_, binding) => {
    expect(bind.missing([binding], {})).toEqual(new Set());
  });
});

describe("bind.commanded", () => {
  it("should read the newest value on the command channel of each state", () => {
    expect(
      bind.commanded(
        [
          { key: "active", source: { type: "channel", key: 1, command: 2 } },
          { key: "open", source: { type: "channel", command: 3 } },
          { key: "idle", source: { type: "channel", key: 1, command: 4 } },
          { key: "level", source: { type: "local", value: 3 } },
        ],
        {
          1: { value: 0, stale: false },
          2: { value: 1, stale: false },
          3: { value: 0, stale: true },
          4: { value: null, stale: false },
        },
      ),
    ).toEqual(
      new Map([
        ["active", 1],
        ["open", 0],
      ]),
    );
  });
});
