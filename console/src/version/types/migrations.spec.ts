// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { migrateSlice, ZERO_SLICE_STATE } from "@/version/types";
import * as v0 from "@/version/types/v0";
import * as v1 from "@/version/types/v1";

describe("migrations", () => {
  describe("slice", () => {
    const STATES = [v0.ZERO_SLICE_STATE, v1.ZERO_SLICE_STATE];
    STATES.forEach((state) =>
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      }),
    );
  });
  describe("slice with a version", () => {
    const consoleVersion = "0.27.0";
    const V0_STATE: v0.SliceState = {
      version: consoleVersion,
    };
    const V1_STATE: v1.SliceState = {
      version: "1.0.0",
      consoleVersion,
      updateNotificationsSilenced: false,
    };
    it(`should migrate slice from ${V0_STATE.version} to latest`, () => {
      const migrated = migrateSlice(V0_STATE);
      expect(migrated).toEqual(migrateSlice(V1_STATE));
    });
  });
});
