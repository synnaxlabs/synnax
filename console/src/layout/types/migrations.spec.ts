// Copyright 2026 Synnax Labs, Inc.
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
import * as v7 from "@/layout/types/v7";
import * as v9 from "@/layout/types/v9";

const STATES: AnySliceState[] = [
  v0.ZERO_SLICE_STATE,
  v1.ZERO_SLICE_STATE,
  v2.ZERO_SLICE_STATE,
  v3.ZERO_SLICE_STATE,
  v4.ZERO_SLICE_STATE,
  v5.ZERO_SLICE_STATE,
  v6.ZERO_SLICE_STATE,
  v7.ZERO_SLICE_STATE,
];

describe("migrations", () => {
  describe("slice", () => {
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });

    it("should drop getStarted layouts and tabs when migrating from v9", () => {
      const stateWithGetStarted: v9.SliceState = {
        ...v9.ZERO_SLICE_STATE,
        layouts: {
          ...v9.ZERO_SLICE_STATE.layouts,
          getStarted: {
            name: "Get Started",
            key: "getStarted",
            location: "mosaic",
            type: "getStarted",
            windowKey: "main",
          },
        },
        mosaics: {
          ...v9.ZERO_SLICE_STATE.mosaics,
          main: {
            ...v9.ZERO_SLICE_STATE.mosaics.main,
            activeTab: "getStarted",
            root: {
              key: 1,
              tabs: [
                {
                  tabKey: "getStarted",
                  name: "Get Started",
                  closable: true,
                  editable: false,
                },
              ],
              selected: "getStarted",
            },
          },
        },
      };
      const migrated = migrateSlice(stateWithGetStarted);
      expect(migrated.layouts.getStarted).toBeUndefined();
      expect(migrated.mosaics.main.activeTab).toBeNull();
      expect(migrated.mosaics.main.root.tabs).toEqual([]);
    });
  });
});
