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
): log.ChannelEntry =>
  log.channelEntryZ.parse({ channel, color: color.ZERO, ...overrides });

const createEmpty = (overrides: Partial<log.Log> = {}): log.Log =>
  log.newZ.parse({ name: "", ...overrides });

const apply = (state: log.Log, ...actions: log.Action[]): log.Log =>
  log.reduceAll(state, actions).next;

const roundTrip = (state: log.Log, ...actions: log.Action[]): log.Log => {
  const { next, inverse } = log.reduceAll(state, actions);
  return log.reduceAll(next, inverse).next;
};

describe("log reducer", () => {
  describe("rename", () => {
    it("should set the log name", () => {
      expect(
        apply(createEmpty({ name: "old" }), log.rename({ name: "new" })).name,
      ).toEqual("new");
    });
    it("should round-trip the previous name through its inverse", () => {
      expect(
        roundTrip(createEmpty({ name: "before" }), log.rename({ name: "after" })).name,
      ).toEqual("before");
    });
    it("should target the log key", () => {
      const state = createEmpty({ key: "11111111-1111-4111-8111-111111111111" });
      expect(log.reduceAll(state, [log.rename({ name: "x" })]).targets).toEqual([
        state.key,
      ]);
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

  describe("setChannels", () => {
    it("should replace the entire ordered list", () => {
      const state = createEmpty({ channels: [createChannelEntry(1)] });
      const out = apply(
        state,
        log.setChannels({ channels: [createChannelEntry(2), createChannelEntry(3)] }),
      );
      expect(out.channels).toEqual([createChannelEntry(2), createChannelEntry(3)]);
    });
    it("should round-trip the previous list through its inverse", () => {
      const state = createEmpty({
        channels: [createChannelEntry(1), createChannelEntry(2)],
      });
      expect(
        roundTrip(state, log.setChannels({ channels: [createChannelEntry(3)] }))
          .channels,
      ).toEqual(state.channels);
    });
    it("should target every channel in the old and new lists", () => {
      const state = createEmpty({ channels: [createChannelEntry(1)] });
      const { targets } = log.reduceAll(state, [
        log.setChannels({ channels: [createChannelEntry(2)] }),
      ]);
      expect(new Set(targets)).toEqual(new Set(["channel:1", "channel:2"]));
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

  describe("setShowChannelNames", () => {
    it("should set the flag and round-trip it", () => {
      const state = createEmpty({ showChannelNames: true });
      expect(
        apply(state, log.setShowChannelNames({ showChannelNames: false }))
          .showChannelNames,
      ).toEqual(false);
      expect(
        roundTrip(state, log.setShowChannelNames({ showChannelNames: false }))
          .showChannelNames,
      ).toEqual(true);
    });
  });

  describe("setShowReceiptTimestamp", () => {
    it("should set the flag and round-trip it", () => {
      const state = createEmpty({ showReceiptTimestamp: true });
      expect(
        apply(state, log.setShowReceiptTimestamp({ showReceiptTimestamp: false }))
          .showReceiptTimestamp,
      ).toEqual(false);
      expect(
        roundTrip(state, log.setShowReceiptTimestamp({ showReceiptTimestamp: false }))
          .showReceiptTimestamp,
      ).toEqual(true);
    });
  });

  describe("log.reduceAll", () => {
    it("should apply actions sequentially", () => {
      const out = apply(
        createEmpty(),
        log.addChannel({ channel: 1 }),
        log.addChannel({ channel: 2 }),
        log.rename({ name: "n" }),
      );
      expect(out.channels.map((e) => e.channel)).toEqual([1, 2]);
      expect(out.name).toEqual("n");
    });
    it("should collect inverses in reverse application order", () => {
      const { inverse } = log.reduceAll(createEmpty(), [
        log.rename({ name: "a" }),
        log.addChannel({ channel: 1 }),
      ]);
      expect(inverse[0].type).toEqual("remove_channel");
      expect(inverse[1].type).toEqual("rename");
    });
    it("should fully invert a multi-action batch", () => {
      const state = createEmpty({ name: "orig", channels: [createChannelEntry(9)] });
      expect(
        roundTrip(
          state,
          log.rename({ name: "next" }),
          log.addChannel({ channel: 1 }),
          log.setShowChannelNames({ showChannelNames: false }),
        ),
      ).toEqual(state);
    });
  });

  describe("wire format", () => {
    it("should parse a known action", () => {
      expect(log.actionZ.parse(log.rename({ name: "x" })).type).toEqual("rename");
    });
    it("should reject an unknown action type", () => {
      expect(() => log.actionZ.parse({ type: "nope" })).toThrow();
    });
    it("should parse a scoped action envelope", () => {
      const parsed = log.scopedActionZ.parse({
        key: "00000000-0000-0000-0000-000000000000",
        dispatchKey: "abc",
        actions: [log.rename({ name: "x" })],
      });
      expect(parsed.seq).toEqual(0);
      expect(parsed.actions[0].type).toEqual("rename");
    });
  });
});
