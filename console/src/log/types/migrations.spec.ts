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

const STATES: AnyState[] = [v0.ZERO_STATE, v1.ZERO_STATE];
const SLICE_STATES: AnySliceState[] = [v0.ZERO_SLICE_STATE, v1.ZERO_SLICE_STATE];

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
    it("should convert v0 channel keys to v1 channel entries with default config", () => {
      const v0State: v0.State = {
        ...v0.ZERO_STATE,
        key: "test",
        channels: [1, 2],
      };
      const migrated = migrateState(v0State);
      expect(migrated.version).toBe(v1.VERSION);
      expect(migrated.channels).toEqual([
        { ...v1.ZERO_CHANNEL_ENTRY, channel: 1 },
        { ...v1.ZERO_CHANNEL_ENTRY, channel: 2 },
      ]);
    });
  });

  describe("anyStateZ", () => {
    it("should parse and migrate v0 state", () => {
      const result = anyStateZ.parse(v0.ZERO_STATE);
      expect(result.version).toBe(v1.VERSION);
    });

    it("should parse v1 state as-is", () => {
      const result = anyStateZ.parse(v1.ZERO_STATE);
      expect(result.version).toBe(v1.VERSION);
    });
  });

  describe("stateFromLog", () => {
    const SERVER_LOG: log.Log = {
      key: "11111111-1111-1111-1111-111111111111",
      name: "Sensor Log",
      channels: [
        { ...v1.ZERO_CHANNEL_ENTRY, channel: 1, color: color.construct("#ff0000") },
        { ...v1.ZERO_CHANNEL_ENTRY, channel: 2, precision: 3 },
      ],
      remoteCreated: true,
      timestampPrecision: 2,
      showChannelNames: false,
      showReceiptTimestamp: false,
    };

    it("should copy persisted fields from the server log", () => {
      const state = v1.stateFromLog(SERVER_LOG);
      expect(state.key).toBe(SERVER_LOG.key);
      expect(state.channels).toEqual(SERVER_LOG.channels);
      expect(state.remoteCreated).toBe(true);
      expect(state.timestampPrecision).toBe(2);
      expect(state.showChannelNames).toBe(false);
      expect(state.showReceiptTimestamp).toBe(false);
    });

    it("should stamp the current state version", () => {
      const state = v1.stateFromLog(SERVER_LOG);
      expect(state.version).toBe(v1.VERSION);
    });

    it("should reset the toolbar to its default state", () => {
      const state = v1.stateFromLog(SERVER_LOG);
      expect(state.toolbar).toEqual(v1.ZERO_TOOLBAR_STATE);
    });

    it("should not carry the server name into the Console state", () => {
      const state = v1.stateFromLog(SERVER_LOG);
      expect(state).not.toHaveProperty("name");
    });

    it("should preserve an empty channels list", () => {
      const state = v1.stateFromLog({ ...SERVER_LOG, channels: [] });
      expect(state.channels).toEqual([]);
    });
  });
});
