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
  ZERO_ANNOTATIONS_STATE,
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

  describe("create", () => {
    it("should fill annotations with the default when absent from a v4 payload", () => {
      const { annotations: _omit, ...withoutAnnotations } = ZERO_STATE;
      const key = "plot-without-annotations";
      // @ts-expect-error: simulate a persisted/imported v4 payload that predates
      // the annotations field. The type forbids this shape, but the runtime
      // reducer must defensively fill in defaults via the zod schema.
      store.dispatch(actions.create({ ...withoutAnnotations, key }));
      expect(store.getState()[SLICE_NAME].plots[key].annotations).toEqual(
        ZERO_ANNOTATIONS_STATE,
      );
    });
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
