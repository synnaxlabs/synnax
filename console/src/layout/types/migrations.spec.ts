// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type AnySliceState, migrateSlice, ZERO_SLICE_STATE } from "@/layout/types";
import * as v0 from "@/layout/types/v0";
import * as v1 from "@/layout/types/v1";
import * as v2 from "@/layout/types/v2";
import * as v3 from "@/layout/types/v3";
import * as v4 from "@/layout/types/v4";
import * as v5 from "@/layout/types/v5";
import * as v6 from "@/layout/types/v6";

const STATES: AnySliceState[] = [
  v0.ZERO_SLICE_STATE,
  v1.ZERO_SLICE_STATE,
  v2.ZERO_SLICE_STATE,
  v3.ZERO_SLICE_STATE,
  v4.ZERO_SLICE_STATE,
  v5.ZERO_SLICE_STATE,
  v6.ZERO_SLICE_STATE,
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
