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
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/lineplot/slice";

describe("Lineplot Slice", () => {
  let store: ReturnType<typeof configureStore<StoreState>>;
  const plotKey = "plot-1";

  beforeEach(() => {
    store = configureStore({
      reducer: { [SLICE_NAME]: reducer },
      preloadedState: { [SLICE_NAME]: ZERO_SLICE_STATE },
    });
    store.dispatch(actions.create({ ...ZERO_STATE, key: plotKey }));
  });

  describe("setRangeAnnotationsVisible", () => {
    it("should default to visible on a newly created plot", () => {
      expect(store.getState()[SLICE_NAME].plots[plotKey].annotations.visible).toBe(
        true,
      );
    });

    it("should hide range annotations", () => {
      store.dispatch(
        actions.setRangeAnnotationsVisible({ key: plotKey, visible: false }),
      );
      expect(store.getState()[SLICE_NAME].plots[plotKey].annotations.visible).toBe(
        false,
      );
    });

    it("should show range annotations after being hidden", () => {
      store.dispatch(
        actions.setRangeAnnotationsVisible({ key: plotKey, visible: false }),
      );
      store.dispatch(
        actions.setRangeAnnotationsVisible({ key: plotKey, visible: true }),
      );
      expect(store.getState()[SLICE_NAME].plots[plotKey].annotations.visible).toBe(
        true,
      );
    });
  });
});
