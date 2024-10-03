// Copyright 2024 Synnax Labs, Inc.
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
  migrateSlice,
  ZERO_SLICE_STATE,
} from "@/layout/migrations";
import * as v0 from "@/layout/migrations/v0";
import * as v3 from "@/layout/migrations/v3";

describe("migrations", () => {
  describe("slice", () => {
    const STATES: AnySliceState[] = [
      v0.ZERO_SLICE_STATE,
      { ...v0.ZERO_SLICE_STATE, version: "0.1.0" },
      { ...v0.ZERO_SLICE_STATE, version: "0.2.0" },
      v3.ZERO_SLICE_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });
});
