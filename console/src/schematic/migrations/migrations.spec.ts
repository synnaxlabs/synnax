// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  migrateSlice,
  migrateState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/migrations";
import * as v0 from "@/schematic/migrations/v0";
import * as v1 from "@/schematic/migrations/v1";
import * as v2 from "@/schematic/migrations/v2";

describe("migrations", () => {
  describe("state", () => {
    const STATES = [v0.ZERO_STATE, v1.ZERO_STATE, v2.ZERO_STATE];
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect({ ...migrated, key: expect.anything() }).toEqual(ZERO_STATE);
      });
    });
  });
  describe("slice", () => {
    const STATES = [v0.ZERO_SLICE_STATE, v1.ZERO_SLICE_STATE, v2.ZERO_SLICE_STATE];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });
});
