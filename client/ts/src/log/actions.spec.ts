// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { log } from "@/log";

const createChannelEntry = (
  channel: number,
  overrides: Partial<log.ChannelEntry> = {},
): log.ChannelEntry => log.channelEntryZ.parse({ channel, ...overrides });

const createEmpty = (overrides: Partial<log.Log> = {}): log.Log =>
  log.logZ.parse({ name: "Test Log", ...overrides });

const apply = (state: log.Log, ...actions: log.Action[]): log.Log =>
  log.reduceAll(state, actions).next;

const roundTrip = (state: log.Log, ...actions: log.Action[]): log.Log => {
  const { next, inverse } = log.reduceAll(state, actions);
  return log.reduceAll(next, inverse).next;
};

const setName = (name: string): log.Action => log.set({ path: ["name"], value: name });

describe("log reducer", () => {
  describe("set of the name", () => {
    it("should set the log name", () => {
      expect(apply(createEmpty({ name: "old" }), setName("new")).name).toEqual("new");
    });
    it("should round-trip the previous name through its inverse", () => {
      expect(roundTrip(createEmpty({ name: "before" }), setName("after")).name).toEqual(
        "before",
      );
    });
    it("should target the name", () => {
      expect(log.reduceAll(createEmpty(), [setName("x")]).targets).toEqual(["name"]);
    });
  });

  describe("addChannel", () => {
    it("should append a default entry for the channel", () => {
      const out = apply(createEmpty(), log.addChannel({ channel: 42 }));
      expect(out.channels).toEqual([createChannelEntry(42)]);
    });
    it("should be a no-op when the channel already has an entry", () => {
      const state = createEmpty({
        channels: [createChannelEntry(42, { alias: "kept" })],
      });
      const out = apply(state, log.addChannel({ channel: 42 }));
      expect(out.channels).toEqual([createChannelEntry(42, { alias: "kept" })]);
    });
    it("should round-trip through removeChannel as its inverse", () => {
      expect(roundTrip(createEmpty(), log.addChannel({ channel: 7 })).channels).toEqual(
        [],
      );
    });
    it("should target the channel", () => {
      expect(
        log.reduceAll(createEmpty(), [log.addChannel({ channel: 42 })]).targets,
      ).toEqual(["channel:42"]);
    });
  });

  describe("removeChannel", () => {
    it("should remove the entry for the channel", () => {
      const state = createEmpty({
        channels: [createChannelEntry(1), createChannelEntry(2)],
      });
      expect(apply(state, log.removeChannel({ channel: 1 })).channels).toEqual([
        createChannelEntry(2),
      ]);
    });
    it("should be a no-op when the channel has no entry", () => {
      const state = createEmpty({ channels: [createChannelEntry(2)] });
      expect(apply(state, log.removeChannel({ channel: 1 })).channels).toEqual([
        createChannelEntry(2),
      ]);
    });
    it("should restore the entry and its config on undo", () => {
      const state = createEmpty({
        channels: [createChannelEntry(1, { alias: "kept", precision: 3 })],
      });
      expect(roundTrip(state, log.removeChannel({ channel: 1 })).channels).toEqual(
        state.channels,
      );
    });
  });

  describe("setChannelEntry", () => {
    it("should insert the entry when the channel is absent", () => {
      const out = apply(
        createEmpty(),
        log.setChannelEntry({ entry: createChannelEntry(5, { alias: "a" }) }),
      );
      expect(out.channels).toEqual([createChannelEntry(5, { alias: "a" })]);
    });
    it("should replace the entry in place when the channel is present", () => {
      const state = createEmpty({
        channels: [createChannelEntry(5, { alias: "old" }), createChannelEntry(6)],
      });
      const out = apply(
        state,
        log.setChannelEntry({ entry: createChannelEntry(5, { alias: "new" }) }),
      );
      expect(out.channels).toEqual([
        createChannelEntry(5, { alias: "new" }),
        createChannelEntry(6),
      ]);
    });
    it("should round-trip an insert through removeChannel", () => {
      expect(
        roundTrip(createEmpty(), log.setChannelEntry({ entry: createChannelEntry(5) }))
          .channels,
      ).toEqual([]);
    });
    it("should round-trip a replace through the previous entry", () => {
      const state = createEmpty({
        channels: [createChannelEntry(5, { precision: 2 })],
      });
      expect(
        roundTrip(
          state,
          log.setChannelEntry({ entry: createChannelEntry(5, { precision: 9 }) }),
        ).channels,
      ).toEqual(state.channels);
    });
    it("should target the channel", () => {
      expect(
        log.reduceAll(createEmpty(), [
          log.setChannelEntry({ entry: createChannelEntry(5) }),
        ]).targets,
      ).toEqual(["channel:5"]);
    });
  });

  describe("per-field channel setters", () => {
    const withEntries = (overrides: Partial<log.ChannelEntry> = {}): log.Log =>
      createEmpty({
        channels: [createChannelEntry(5, overrides), createChannelEntry(6)],
      });

    interface FieldCase {
      name: string;
      action: log.Action;
      read: (entry: log.ChannelEntry) => unknown;
      expected: unknown;
    }

    const red = color.construct("#ff0000");
    const cases: FieldCase[] = [
      {
        name: "setChannelColor",
        action: log.setChannelColor({ channel: 5, color: red }),
        read: (e) => e.color,
        expected: red,
      },
      {
        name: "setChannelNotation",
        action: log.setChannelNotation({ channel: 5, notation: "scientific" }),
        read: (e) => e.notation,
        expected: "scientific",
      },
      {
        name: "setChannelPrecision",
        action: log.setChannelPrecision({ channel: 5, precision: 4 }),
        read: (e) => e.precision,
        expected: 4,
      },
      {
        name: "setChannelAlias",
        action: log.setChannelAlias({ channel: 5, alias: "volts" }),
        read: (e) => e.alias,
        expected: "volts",
      },
      {
        name: "setChannelTimestampFormat",
        action: log.setChannelTimestampFormat({ channel: 5, format: "time" }),
        read: (e) => e.timestampConfig.format,
        expected: "time",
      },
      {
        name: "setChannelTimestampTimeZone",
        action: log.setChannelTimestampTimeZone({ channel: 5, timeZone: "UTC" }),
        read: (e) => e.timestampConfig.timeZone,
        expected: "UTC",
      },
    ];

    it.each(cases)(
      "$name sets the field on the matching entry",
      ({ action, read, expected }) => {
        const out = apply(withEntries(), action);
        expect(read(out.channels[0])).toEqual(expected);
        expect(out.channels[1]).toEqual(createChannelEntry(6));
      },
    );

    it.each(cases)(
      "$name round-trips the old value through its inverse",
      ({ action, read }) => {
        const state = withEntries();
        const before = read(state.channels[0]);
        expect(read(roundTrip(state, action).channels[0])).toEqual(before);
      },
    );

    it.each(cases)("$name targets the channel", ({ action }) => {
      expect(log.reduceAll(withEntries(), [action]).targets).toEqual(["channel:5"]);
    });

    it.each(cases)("$name keys its undoable by channel", ({ action }) => {
      expect(log.kindOf([action])).toEqual("channel:5");
    });

    it.each(cases)("$name is a no-op when the channel is absent", ({ action }) => {
      const state = createEmpty({ channels: [createChannelEntry(6)] });
      expect(apply(state, action).channels).toEqual([createChannelEntry(6)]);
    });

    it("should clear a channel color set to absent and restore it on undo", () => {
      const state = createEmpty({ channels: [createChannelEntry(5, { color: red })] });
      const cleared = apply(state, log.setChannelColor({ channel: 5 }));
      expect(cleared.channels[0].color).toBeUndefined();
      const restored = roundTrip(state, log.setChannelColor({ channel: 5 }));
      expect(restored.channels[0].color).toEqual(red);
    });
  });

  describe("swapChannel", () => {
    it("should repoint a channel in place, keeping position and config", () => {
      const state = createEmpty({
        channels: [
          createChannelEntry(1),
          createChannelEntry(2, { alias: "kept", precision: 3 }),
          createChannelEntry(4),
        ],
      });
      const out = apply(state, log.swapChannel({ from: 2, to: 5 }));
      expect(out.channels.map((e) => e.channel)).toEqual([1, 5, 4]);
      expect(out.channels[1]).toEqual(
        createChannelEntry(5, { alias: "kept", precision: 3 }),
      );
    });
    it("should be a no-op when the from channel is absent", () => {
      const state = createEmpty({ channels: [createChannelEntry(1)] });
      expect(apply(state, log.swapChannel({ from: 99, to: 5 })).channels).toEqual([
        createChannelEntry(1),
      ]);
    });
    it("should be a no-op when the to channel has an entry", () => {
      const channels = [
        createChannelEntry(1, { alias: "one" }),
        createChannelEntry(2, { alias: "two" }),
      ];
      const state = createEmpty({ channels });
      const { next, inverse } = log.reduceAll(state, [
        log.swapChannel({ from: 1, to: 2 }),
      ]);
      expect(next.channels).toEqual(channels);
      expect(inverse).toEqual([]);
    });
    it("should round-trip through the reverse swap", () => {
      const state = createEmpty({
        channels: [createChannelEntry(1), createChannelEntry(2)],
      });
      expect(roundTrip(state, log.swapChannel({ from: 2, to: 5 })).channels).toEqual(
        state.channels,
      );
    });
    it("should target both the from and to channels", () => {
      const state = createEmpty({ channels: [createChannelEntry(2)] });
      const { targets } = log.reduceAll(state, [log.swapChannel({ from: 2, to: 5 })]);
      expect(new Set(targets)).toEqual(new Set(["channel:2", "channel:5"]));
    });
  });

  describe("setChannelPrecision", () => {
    it("should clear the precision and round-trip it", () => {
      const state = createEmpty({
        channels: [createChannelEntry(5, { precision: 2 })],
      });
      const action = log.setChannelPrecision({ channel: 5 });
      expect(apply(state, action).channels[0].precision).toBeUndefined();
      expect(roundTrip(state, action).channels[0].precision).toEqual(2);
    });
  });

  describe("setTimestampPrecision", () => {
    it("should set the precision", () => {
      expect(
        apply(createEmpty(), log.setTimestampPrecision({ timestampPrecision: 3 }))
          .timestampPrecision,
      ).toEqual(3);
    });
    it("should round-trip the previous precision", () => {
      const state = createEmpty({ timestampPrecision: 1 });
      expect(
        roundTrip(state, log.setTimestampPrecision({ timestampPrecision: 2 }))
          .timestampPrecision,
      ).toEqual(1);
    });
  });

  describe("setChannelNamesHidden", () => {
    it("should set the flag and round-trip it", () => {
      const state = createEmpty({ channelNamesHidden: true });
      expect(
        apply(state, log.setChannelNamesHidden({ channelNamesHidden: false }))
          .channelNamesHidden,
      ).toEqual(false);
      expect(
        roundTrip(state, log.setChannelNamesHidden({ channelNamesHidden: false }))
          .channelNamesHidden,
      ).toEqual(true);
    });
  });

  describe("setTimestampsHidden", () => {
    it("should set the flag and round-trip it", () => {
      const state = createEmpty({ timestampsHidden: true });
      expect(
        apply(state, log.setTimestampsHidden({ timestampsHidden: false }))
          .timestampsHidden,
      ).toEqual(false);
      expect(
        roundTrip(state, log.setTimestampsHidden({ timestampsHidden: false }))
          .timestampsHidden,
      ).toEqual(true);
    });
  });

  describe("setTimeZone", () => {
    it("should set the time zone and round-trip it", () => {
      const state = createEmpty({ timeZone: "local" });
      expect(apply(state, log.setTimeZone({ timeZone: "UTC" })).timeZone).toEqual(
        "UTC",
      );
      expect(roundTrip(state, log.setTimeZone({ timeZone: "UTC" })).timeZone).toEqual(
        "local",
      );
    });
    it("should target the log key", () => {
      const state = createEmpty({ key: "11111111-1111-4111-8111-111111111111" });
      expect(
        log.reduceAll(state, [log.setTimeZone({ timeZone: "UTC" })]).targets,
      ).toEqual([state.key]);
    });
  });

  describe("log.reduceAll", () => {
    it("should apply actions sequentially", () => {
      const out = apply(
        createEmpty(),
        log.addChannel({ channel: 1 }),
        log.addChannel({ channel: 2 }),
        setName("n"),
      );
      expect(out.channels.map((e) => e.channel)).toEqual([1, 2]);
      expect(out.name).toEqual("n");
    });
    it("should collect inverses in reverse application order", () => {
      const { inverse } = log.reduceAll(createEmpty(), [
        setName("a"),
        log.addChannel({ channel: 1 }),
      ]);
      expect(inverse[0].type).toEqual("remove_channel");
      expect(inverse[1].type).toEqual("set");
    });
    it("should fully invert a multi-action batch", () => {
      const state = createEmpty({ name: "orig", channels: [createChannelEntry(9)] });
      expect(
        roundTrip(
          state,
          setName("next"),
          log.addChannel({ channel: 1 }),
          log.setChannelNamesHidden({ channelNamesHidden: false }),
        ),
      ).toEqual(state);
    });
  });

  describe("wire format", () => {
    it("should parse a known action", () => {
      expect(log.actionZ.parse(setName("x")).type).toEqual("set");
    });
    it("should reject an unknown action type", () => {
      expect(() => log.actionZ.parse({ type: "nope" })).toThrow();
    });
    it("should parse a scoped action envelope", () => {
      const parsed = log.scopedActionZ.parse({
        key: "00000000-0000-0000-0000-000000000000",
        dispatchKey: "abc",
        actions: [setName("x")],
      });
      expect(parsed.seq).toEqual(0);
      expect(parsed.actions[0].type).toEqual("set");
    });
  });
});

describe("group and project patches", () => {
  const G1 = "11111111-1111-4111-8111-111111111111";
  const G2 = "22222222-2222-4222-8222-222222222222";
  it("should move the document into the group", () => {
    const doc = createEmpty();
    expect(
      log.reduceAll(doc, [log.set({ path: ["group"], value: G1 })]).next.group,
    ).toEqual(G1);
  });
  it("should move the document out of its group", () => {
    const doc = createEmpty({ group: G1 });
    expect(
      log.reduceAll(doc, [log.remove({ key: "group" })]).next.group,
    ).toBeUndefined();
  });
  it("should restore the previous group through its inverse", () => {
    const doc = createEmpty({ group: G1 });
    const { next, inverse } = log.reduceAll(doc, [
      log.set({ path: ["group"], value: G2 }),
    ]);
    expect(log.reduceAll(next, inverse).next.group).toEqual(G1);
  });
  it("should move the document into the project", () => {
    const doc = createEmpty({ project: G1 });
    expect(
      log.reduceAll(doc, [log.set({ path: ["project"], value: G2 })]).next.project,
    ).toEqual(G2);
  });
});
