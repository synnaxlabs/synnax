// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it } from "vitest";

import {
  actions,
  reducer,
  SLICE_NAME,
  type StoreState,
  ZERO_CHANNEL_CONFIG,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/log/slice";
import { stateZ } from "@/log/types";
import { channelConfigZ } from "@/log/types/v0";

const ch = (channel: number) => ({ ...ZERO_CHANNEL_CONFIG, channel });

describe("Log Slice", () => {
  let store: ReturnType<typeof configureStore<StoreState>>;

  beforeEach(() => {
    store = configureStore({
      reducer: { [SLICE_NAME]: reducer },
      preloadedState: { [SLICE_NAME]: ZERO_SLICE_STATE },
    });
  });

  describe("create", () => {
    it("should create a new log", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      const state = store.getState()[SLICE_NAME];
      expect(state.logs[key]).toBeDefined();
      expect(state.logs[key].key).toBe(key);
      expect(state.logs[key].channels).toEqual([]);
    });

    it("should create multiple logs independently", () => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: "log-1" }));
      store.dispatch(actions.create({ ...ZERO_STATE, key: "log-2" }));
      const state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.logs)).toHaveLength(2);
    });
  });

  describe("setTimestampPrecision", () => {
    it("should update timestamp precision", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      store.dispatch(actions.setTimestampPrecision({ key, timestampPrecision: 2 }));
      expect(store.getState()[SLICE_NAME].logs[key].timestampPrecision).toBe(2);
    });
  });

  describe("setChannelConfig", () => {
    it("should update config for a channel entry", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key, channels: [ch(42)] }));
      store.dispatch(
        actions.setChannelConfig({ key, channelKey: 42, config: { color: "#ff0000" } }),
      );
      const entry = store.getState()[SLICE_NAME].logs[key].channels[0];
      expect(entry).toEqual({ ...ZERO_CHANNEL_CONFIG, channel: 42, color: "#ff0000" });
    });

    it("should merge partial updates into an existing config", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key, channels: [ch(1)] }));
      store.dispatch(
        actions.setChannelConfig({
          key,
          channelKey: 1,
          config: { color: "#ff0000", precision: 2 },
        }),
      );
      store.dispatch(
        actions.setChannelConfig({ key, channelKey: 1, config: { precision: 4 } }),
      );
      const entry = store.getState()[SLICE_NAME].logs[key].channels[0];
      expect(entry).toEqual({
        channel: 1,
        color: "#ff0000",
        notation: "standard",
        precision: 4,
        alias: "",
      });
    });
  });

  describe("setShowChannelNames", () => {
    it("should update showChannelNames", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      expect(store.getState()[SLICE_NAME].logs[key].showChannelNames).toBe(true);
      store.dispatch(actions.setShowChannelNames({ key, showChannelNames: false }));
      expect(store.getState()[SLICE_NAME].logs[key].showChannelNames).toBe(false);
    });
  });

  describe("addChannel", () => {
    it("should append a channel entry to the list", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      store.dispatch(actions.addChannel({ key, channelKey: 10 }));
      store.dispatch(actions.addChannel({ key, channelKey: 20 }));
      const channels = store.getState()[SLICE_NAME].logs[key].channels;
      expect(channels).toEqual([ch(10), ch(20)]);
    });
  });

  describe("removeChannelByIndex", () => {
    it("should remove the channel at the given index", () => {
      const key = "log-1";
      store.dispatch(
        actions.create({ ...ZERO_STATE, key, channels: [ch(1), ch(2), ch(3)] }),
      );
      store.dispatch(actions.removeChannelByIndex({ key, index: 1 }));
      const channels = store.getState()[SLICE_NAME].logs[key].channels;
      expect(channels).toEqual([ch(1), ch(3)]);
    });
  });

  describe("setChannelAtIndex", () => {
    it("should replace the channel key at the given index", () => {
      const key = "log-1";
      store.dispatch(
        actions.create({ ...ZERO_STATE, key, channels: [ch(1), ch(2), ch(3)] }),
      );
      store.dispatch(actions.setChannelAtIndex({ key, index: 1, channelKey: 99 }));
      const channels = store.getState()[SLICE_NAME].logs[key].channels;
      expect(channels).toEqual([ch(1), ch(99), ch(3)]);
    });
  });

  describe("setRemoteCreated", () => {
    it("should mark the log as remotely created", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      expect(store.getState()[SLICE_NAME].logs[key].remoteCreated).toBe(false);
      store.dispatch(actions.setRemoteCreated({ key }));
      expect(store.getState()[SLICE_NAME].logs[key].remoteCreated).toBe(true);
    });
  });

  describe("remove", () => {
    it("should remove a log by key", () => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: "log-1" }));
      store.dispatch(actions.remove({ keys: ["log-1"] }));
      expect(store.getState()[SLICE_NAME].logs["log-1"]).toBeUndefined();
    });

    it("should remove multiple logs at once", () => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: "log-1" }));
      store.dispatch(actions.create({ ...ZERO_STATE, key: "log-2" }));
      store.dispatch(actions.remove({ keys: ["log-1", "log-2"] }));
      expect(Object.keys(store.getState()[SLICE_NAME].logs)).toHaveLength(0);
    });
  });

  describe("stateZ schema", () => {
    it("should accept a valid state", () => {
      expect(() => stateZ.parse(ZERO_STATE)).not.toThrow();
    });

    it("should reject timestampPrecision above 3", () => {
      expect(() => stateZ.parse({ ...ZERO_STATE, timestampPrecision: 4 })).toThrow();
    });

    it("should reject timestampPrecision below 0", () => {
      expect(() => stateZ.parse({ ...ZERO_STATE, timestampPrecision: -1 })).toThrow();
    });

    it("should reject channel precision above 17", () => {
      expect(() => channelConfigZ.parse({ color: "", precision: 18 })).toThrow();
    });

    it("should reject channel precision below -1", () => {
      expect(() => channelConfigZ.parse({ color: "", precision: -2 })).toThrow();
    });

    it("should default showChannelNames to true when missing", () => {
      const { showChannelNames: _, ...withoutField } = ZERO_STATE;
      const parsed = stateZ.parse(withoutField);
      expect(parsed.showChannelNames).toBe(true);
    });
  });
});
