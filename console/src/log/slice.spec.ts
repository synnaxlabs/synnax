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
import { channelConfigZ, stateZ } from "@/log/types/v0";

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

  describe("setChannels", () => {
    it("should update the channel list", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      store.dispatch(actions.setChannels({ key, channels: [1, 2, 3] }));
      expect(store.getState()[SLICE_NAME].logs[key].channels).toEqual([1, 2, 3]);
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
    it("should create config for a new channel key", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
      store.dispatch(
        actions.setChannelConfig({ key, channelKey: 42, config: { color: "#ff0000" } }),
      );
      expect(store.getState()[SLICE_NAME].logs[key].channelConfigs["42"]).toEqual({
        ...ZERO_CHANNEL_CONFIG,
        color: "#ff0000",
      });
    });

    it("should merge partial updates into an existing config", () => {
      const key = "log-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));
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
      expect(store.getState()[SLICE_NAME].logs[key].channelConfigs["1"]).toEqual({
        color: "#ff0000",
        precision: 4,
      });
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
  });
});
