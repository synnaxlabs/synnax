// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
        { channel: 1, ...v1.ZERO_CHANNEL_CONFIG },
        { channel: 2, ...v1.ZERO_CHANNEL_CONFIG },
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
});
