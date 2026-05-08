// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { describe, expect, it } from "vitest";

import { MIDDLEWARE } from "@/arc/middleware";
import {
  actions,
  reducer,
  SLICE_NAME,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/arc/slice";
import { Layout } from "@/layout";

describe("Arc Middleware", () => {
  const layoutKey = "arc-1";
  const otherKey = "arc-2";
  const modalKey = "modal-1";

  const buildStore = (
    layouts: Record<string, Layout.State>,
    activeTab: string | null,
  ) => {
    const store = configureStore({
      reducer: combineReducers({
        [SLICE_NAME]: reducer,
        [Layout.SLICE_NAME]: Layout.reducer,
        drift: Drift.reducer,
      }),
      preloadedState: {
        [SLICE_NAME]: ZERO_SLICE_STATE,
        [Layout.SLICE_NAME]: {
          ...Layout.ZERO_SLICE_STATE,
          layouts: { ...Layout.ZERO_SLICE_STATE.layouts, ...layouts },
          mosaics: {
            ...Layout.ZERO_SLICE_STATE.mosaics,
            [MAIN_WINDOW]: {
              ...Layout.ZERO_SLICE_STATE.mosaics[MAIN_WINDOW],
              activeTab,
            },
          },
        },
      },
      middleware: (getDefault) => getDefault().concat(MIDDLEWARE),
    });
    store.dispatch(actions.create({ ...ZERO_STATE, key: layoutKey }));
    return store;
  };

  const mosaicLayout = (key: string): Layout.State => ({
    key,
    windowKey: MAIN_WINDOW,
    type: "arc",
    name: key,
    location: "mosaic",
  });

  const modalLayout = (key: string): Layout.State => ({
    key,
    windowKey: MAIN_WINDOW,
    type: "arc",
    name: key,
    location: "modal",
  });

  it("should apply setSelected when the arc is the active mosaic tab", () => {
    const store = buildStore({ [layoutKey]: mosaicLayout(layoutKey) }, layoutKey);
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    expect(store.getState()[SLICE_NAME].arcs[layoutKey].graph.selected).toEqual(["n1"]);
  });

  it("should drop setSelected when a modal is open in the arc's window", () => {
    const store = buildStore(
      {
        [layoutKey]: mosaicLayout(layoutKey),
        [modalKey]: modalLayout(modalKey),
      },
      layoutKey,
    );
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    expect(store.getState()[SLICE_NAME].arcs[layoutKey].graph.selected).toEqual([]);
  });

  it("should drop setSelected when a different mosaic tab is active", () => {
    const store = buildStore(
      {
        [layoutKey]: mosaicLayout(layoutKey),
        [otherKey]: mosaicLayout(otherKey),
      },
      otherKey,
    );
    store.dispatch(actions.setSelected({ key: layoutKey, selected: ["n1"] }));
    expect(store.getState()[SLICE_NAME].arcs[layoutKey].graph.selected).toEqual([]);
  });
});
