// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  type AnySliceState,
  type AnyState,
  anyStateZ,
  migrateSlice,
  migrateState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/log/types";
import * as v0 from "@/log/types/v0";
import * as v1 from "@/log/types/v1";
import * as v2 from "@/log/types/v2";

const STATES: AnyState[] = [v0.ZERO_STATE, v1.ZERO_STATE, v2.ZERO_STATE];
const SLICE_STATES: AnySliceState[] = [
  v0.ZERO_SLICE_STATE,
  v1.ZERO_SLICE_STATE,
  v2.ZERO_SLICE_STATE,
];

describe("log type migrations", () => {
  describe("state", () => {
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect(migrated).toEqual(ZERO_STATE);
      });
    });
  });

  describe("slice", () => {
    SLICE_STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });

  describe("state migration preserves data", () => {
    it("should convert v0 channel keys to v2 channel entries with default config", () => {
      const v0State: v0.State = {
        ...v0.ZERO_STATE,
        key: "test",
        channels: [1, 2],
      };
      const migrated = migrateState(v0State);
      expect(migrated.version).toBe(v2.VERSION);
      expect(migrated.channels).toEqual([
        { ...v2.ZERO_CHANNEL_ENTRY, channel: 1 },
        { ...v2.ZERO_CHANNEL_ENTRY, channel: 2 },
      ]);
    });
  });

  describe("v1 -> v2 color migration", () => {
    const v1StateWithColor = (channelColor: string): v1.State => ({
      ...v1.ZERO_STATE,
      key: "test",
      channels: [{ ...v1.ZERO_CHANNEL_ENTRY, channel: 1, color: channelColor }],
    });

    it("should convert an empty-string color to color.ZERO", () => {
      const migrated = migrateState(v1StateWithColor(""));
      expect(migrated.version).toBe(v2.VERSION);
      expect(migrated.channels[0].color).toEqual(color.ZERO);
    });

    it("should convert a hex string color to a real color", () => {
      const migrated = migrateState(v1StateWithColor("#ff0000"));
      expect(migrated.channels[0].color).toEqual(color.construct("#ff0000"));
    });

    it("should fall back to color.ZERO for an unparseable color", () => {
      const migrated = migrateState(v1StateWithColor("not-a-color"));
      expect(migrated.channels[0].color).toEqual(color.ZERO);
    });

    it("should migrate empty-string colors when loading a persisted slice", () => {
      const slice: v1.SliceState = {
        ...v1.ZERO_SLICE_STATE,
        logs: { test: v1StateWithColor("") },
      };
      const migrated = migrateSlice(slice);
      expect(migrated.version).toBe(v2.VERSION);
      expect(migrated.logs.test.channels[0].color).toEqual(color.ZERO);
    });
  });

  describe("anyStateZ", () => {
    it("should parse and migrate v0 state", () => {
      const result = anyStateZ.parse(v0.ZERO_STATE);
      expect(result.version).toBe(v2.VERSION);
    });

    it("should parse and migrate v1 state with an empty-string color", () => {
      const result = anyStateZ.parse({
        ...v1.ZERO_STATE,
        channels: [{ ...v1.ZERO_CHANNEL_ENTRY, channel: 1, color: "" }],
      });
      expect(result.version).toBe(v2.VERSION);
      expect(result.channels[0].color).toEqual(color.ZERO);
    });

    it("should parse v2 state as-is", () => {
      const result = anyStateZ.parse(v2.ZERO_STATE);
      expect(result.version).toBe(v2.VERSION);
    });

    it("should parse a persisted v1 log export with an empty color and no timestamp", () => {
      const result = anyStateZ.parse({
        key: "424ef02f-6ec3-4af6-bdd2-242964a747d8",
        version: "1.0.0",
        channels: [
          {
            channel: 1048581,
            color: "",
            notation: "standard",
            precision: -1,
            alias: "",
          },
        ],
        remoteCreated: true,
        timestampPrecision: 0,
        showChannelNames: true,
        showReceiptTimestamp: true,
        toolbar: { activeTab: "channels" },
        type: "log",
      });
      expect(result.version).toBe(v2.VERSION);
      expect(result.channels[0].color).toEqual(color.ZERO);
      expect(result.channels[0].timestamp).toEqual({
        format: "preciseDate",
        tz: "local",
      });
    });
  });

  describe("stateFromLog", () => {
    const SERVER_LOG: log.Log = {
      key: "11111111-1111-1111-1111-111111111111",
      name: "Sensor Log",
      channels: [
        { ...v2.ZERO_CHANNEL_ENTRY, channel: 1, color: color.construct("#ff0000") },
        { ...v2.ZERO_CHANNEL_ENTRY, channel: 2, precision: 3 },
      ],
      remoteCreated: true,
      timestampPrecision: 2,
      showChannelNames: false,
      showReceiptTimestamp: false,
    };

    it("should copy persisted fields from the server log", () => {
      const state = v2.stateFromLog(SERVER_LOG);
      expect(state.key).toBe(SERVER_LOG.key);
      expect(state.channels).toEqual(SERVER_LOG.channels);
      expect(state.remoteCreated).toBe(true);
      expect(state.timestampPrecision).toBe(2);
      expect(state.showChannelNames).toBe(false);
      expect(state.showReceiptTimestamp).toBe(false);
    });

    it("should stamp the current state version", () => {
      const state = v2.stateFromLog(SERVER_LOG);
      expect(state.version).toBe(v2.VERSION);
    });

    it("should reset the toolbar to its default state", () => {
      const state = v2.stateFromLog(SERVER_LOG);
      expect(state.toolbar).toEqual(v2.ZERO_TOOLBAR_STATE);
    });

    it("should not carry the server name into the Console state", () => {
      const state = v2.stateFromLog(SERVER_LOG);
      expect(state).not.toHaveProperty("name");
    });

    it("should preserve an empty channels list", () => {
      const state = v2.stateFromLog({ ...SERVER_LOG, channels: [] });
      expect(state.channels).toEqual([]);
    });
  });
});
