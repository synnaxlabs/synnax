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

import { reduceAll } from "@/log/actions";
import {
  type Action,
  actionZ,
  addChannel,
  removeChannel,
  rename,
  scopedActionZ,
  setChannelEntry,
  setChannels,
  setShowChannelNames,
  setShowReceiptTimestamp,
  setTimestampPrecision,
} from "@/log/actions.gen";
import {
  type ChannelEntry,
  channelEntryZ,
  type Log,
  logZ,
} from "@/log/types.gen";

const entry = (channel: number, overrides: Partial<ChannelEntry> = {}): ChannelEntry =>
  channelEntryZ.parse({ channel, color: color.ZERO, ...overrides });

const empty = (overrides: Partial<Log> = {}): Log =>
  logZ.parse({ key: "00000000-0000-0000-0000-000000000000", name: "", ...overrides });

const apply = (state: Log, ...actions: Action[]): Log => reduceAll(state, actions).next;

// roundTrip applies actions then applies their captured inverses in reverse. The
// result must equal the input state for actions that fully invert.
const roundTrip = (state: Log, ...actions: Action[]): Log => {
  const { next, inverse } = reduceAll(state, actions);
  return reduceAll(next, inverse).next;
};

describe("log reducer", () => {
  describe("rename", () => {
    it("should set the log name", () => {
      expect(apply(empty({ name: "old" }), rename({ name: "new" })).name).toEqual(
        "new",
      );
    });
    it("should round-trip the previous name through its inverse", () => {
      expect(
        roundTrip(empty({ name: "before" }), rename({ name: "after" })).name,
      ).toEqual("before");
    });
    it("should target the log key", () => {
      const state = empty({ key: "11111111-1111-4111-8111-111111111111" });
      expect(reduceAll(state, [rename({ name: "x" })]).targets).toEqual([state.key]);
    });
  });

  describe("addChannel", () => {
    it("should append a default entry for the channel", () => {
      const out = apply(empty(), addChannel({ channel: 42 }));
      expect(out.channels).toEqual([entry(42)]);
    });
    it("should be a no-op when the channel already has an entry", () => {
      const state = empty({ channels: [entry(42, { alias: "kept" })] });
      const out = apply(state, addChannel({ channel: 42 }));
      expect(out.channels).toEqual([entry(42, { alias: "kept" })]);
    });
    it("should round-trip through removeChannel as its inverse", () => {
      expect(roundTrip(empty(), addChannel({ channel: 7 })).channels).toEqual([]);
    });
    it("should target the channel", () => {
      expect(reduceAll(empty(), [addChannel({ channel: 42 })]).targets).toEqual([
        "channel:42",
      ]);
    });
  });

  describe("removeChannel", () => {
    it("should remove the entry for the channel", () => {
      const state = empty({ channels: [entry(1), entry(2)] });
      expect(apply(state, removeChannel({ channel: 1 })).channels).toEqual([entry(2)]);
    });
    it("should be a no-op when the channel has no entry", () => {
      const state = empty({ channels: [entry(2)] });
      expect(apply(state, removeChannel({ channel: 1 })).channels).toEqual([entry(2)]);
    });
    it("should restore the entry and its config on undo", () => {
      const state = empty({ channels: [entry(1, { alias: "kept", precision: 3 })] });
      expect(roundTrip(state, removeChannel({ channel: 1 })).channels).toEqual(
        state.channels,
      );
    });
  });

  describe("setChannelEntry", () => {
    it("should insert the entry when the channel is absent", () => {
      const out = apply(empty(), setChannelEntry({ entry: entry(5, { alias: "a" }) }));
      expect(out.channels).toEqual([entry(5, { alias: "a" })]);
    });
    it("should replace the entry in place when the channel is present", () => {
      const state = empty({ channels: [entry(5, { alias: "old" }), entry(6)] });
      const out = apply(state, setChannelEntry({ entry: entry(5, { alias: "new" }) }));
      expect(out.channels).toEqual([entry(5, { alias: "new" }), entry(6)]);
    });
    it("should round-trip an insert through removeChannel", () => {
      expect(roundTrip(empty(), setChannelEntry({ entry: entry(5) })).channels).toEqual(
        [],
      );
    });
    it("should round-trip a replace through the previous entry", () => {
      const state = empty({ channels: [entry(5, { precision: 2 })] });
      expect(
        roundTrip(state, setChannelEntry({ entry: entry(5, { precision: 9 }) }))
          .channels,
      ).toEqual(state.channels);
    });
    it("should target the channel", () => {
      expect(
        reduceAll(empty(), [setChannelEntry({ entry: entry(5) })]).targets,
      ).toEqual(["channel:5"]);
    });
  });

  describe("setChannels", () => {
    it("should replace the entire ordered list", () => {
      const state = empty({ channels: [entry(1)] });
      const out = apply(state, setChannels({ channels: [entry(2), entry(3)] }));
      expect(out.channels).toEqual([entry(2), entry(3)]);
    });
    it("should round-trip the previous list through its inverse", () => {
      const state = empty({ channels: [entry(1), entry(2)] });
      expect(roundTrip(state, setChannels({ channels: [entry(3)] })).channels).toEqual(
        state.channels,
      );
    });
    it("should target every channel in the old and new lists", () => {
      const state = empty({ channels: [entry(1)] });
      const { targets } = reduceAll(state, [setChannels({ channels: [entry(2)] })]);
      expect(new Set(targets)).toEqual(new Set(["channel:1", "channel:2"]));
    });
  });

  describe("setTimestampPrecision", () => {
    it("should set the precision", () => {
      expect(
        apply(empty(), setTimestampPrecision({ timestampPrecision: 3 }))
          .timestampPrecision,
      ).toEqual(3);
    });
    it("should round-trip the previous precision", () => {
      const state = empty({ timestampPrecision: 1 });
      expect(
        roundTrip(state, setTimestampPrecision({ timestampPrecision: 2 }))
          .timestampPrecision,
      ).toEqual(1);
    });
  });

  describe("setShowChannelNames", () => {
    it("should set the flag and round-trip it", () => {
      const state = empty({ showChannelNames: true });
      expect(
        apply(state, setShowChannelNames({ showChannelNames: false })).showChannelNames,
      ).toEqual(false);
      expect(
        roundTrip(state, setShowChannelNames({ showChannelNames: false }))
          .showChannelNames,
      ).toEqual(true);
    });
  });

  describe("setShowReceiptTimestamp", () => {
    it("should set the flag and round-trip it", () => {
      const state = empty({ showReceiptTimestamp: true });
      expect(
        apply(state, setShowReceiptTimestamp({ showReceiptTimestamp: false }))
          .showReceiptTimestamp,
      ).toEqual(false);
      expect(
        roundTrip(state, setShowReceiptTimestamp({ showReceiptTimestamp: false }))
          .showReceiptTimestamp,
      ).toEqual(true);
    });
  });

  describe("reduceAll", () => {
    it("should apply actions sequentially", () => {
      const out = apply(
        empty(),
        addChannel({ channel: 1 }),
        addChannel({ channel: 2 }),
        rename({ name: "n" }),
      );
      expect(out.channels.map((e) => e.channel)).toEqual([1, 2]);
      expect(out.name).toEqual("n");
    });
    it("should collect inverses in reverse application order", () => {
      const { inverse } = reduceAll(empty(), [
        rename({ name: "a" }),
        addChannel({ channel: 1 }),
      ]);
      expect(inverse[0].type).toEqual("remove_channel");
      expect(inverse[1].type).toEqual("rename");
    });
    it("should fully invert a multi-action batch", () => {
      const state = empty({ name: "orig", channels: [entry(9)] });
      expect(
        roundTrip(
          state,
          rename({ name: "next" }),
          addChannel({ channel: 1 }),
          setShowChannelNames({ showChannelNames: false }),
        ),
      ).toEqual(state);
    });
  });

  describe("wire format", () => {
    it("should parse a known action", () => {
      expect(actionZ.parse(rename({ name: "x" })).type).toEqual("rename");
    });
    it("should reject an unknown action type", () => {
      expect(() => actionZ.parse({ type: "nope" })).toThrow();
    });
    it("should parse a scoped action envelope", () => {
      const parsed = scopedActionZ.parse({
        key: "00000000-0000-0000-0000-000000000000",
        dispatchKey: "abc",
        actions: [rename({ name: "x" })],
      });
      expect(parsed.seq).toEqual(0);
      expect(parsed.actions[0].type).toEqual("rename");
    });
  });
});
