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
import * as v1 from "@/layout/migrations/v1";
import * as v2 from "@/layout/migrations/v2";
import * as v3 from "@/layout/migrations/v3";
import * as v4 from "@/layout/migrations/v4";

const STATES: AnySliceState[] = [
  v0.ZERO_SLICE_STATE,
  v1.ZERO_SLICE_STATE,
  v2.ZERO_SLICE_STATE,
  v3.ZERO_SLICE_STATE,
  v4.ZERO_SLICE_STATE,
];

describe("migrations", () => {
  describe("slice", () => {
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });
});
