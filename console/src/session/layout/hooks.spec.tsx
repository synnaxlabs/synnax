// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { renderHook } from "@testing-library/react";
import { act, type PropsWithChildren } from "react";
import { Provider, useStore } from "react-redux";
import { describe, expect, it } from "vitest";

import { Layout } from "@/session/layout";
import { select } from "@/session/layout/selectors";
import { Modals } from "@/session/modals";

describe("layout hooks", () => {
  describe("placing & removing", () => {
    it("should place a layout within the store", () => {
      const store = configureStore({
        reducer: combineReducers({
          [Layout.SLICE_NAME]: Layout.reducer,
          [Drift.SLICE_NAME]: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>
          <Modals.Provider>{children}</Modals.Provider>
        </Provider>
      );
      const { result } = renderHook(
        () => ({
          placer: Layout.usePlacer(),
          store: useStore(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.placer({
          key: "test",
          location: "mosaic",
          type: "cat",
          name: "test",
          window: {
            title: "test",
          },
        });
      });
      const state = select(store.getState(), "test");
      expect(state).toBeDefined();
      expect(state?.key).toBe("test");
      expect(state?.location).toBe("mosaic");
      expect(state?.type).toBe("cat");
      expect(state?.name).toBe("test");
    });

    it("should remove a layout from the store", () => {
      const store = configureStore({
        reducer: combineReducers({
          [Layout.SLICE_NAME]: Layout.reducer,
          [Drift.SLICE_NAME]: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>
          <Modals.Provider>{children}</Modals.Provider>
        </Provider>
      );
      const { result } = renderHook(
        () => ({
          placer: Layout.usePlacer(),
          store: useStore(),
          remover: Layout.useRemover(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.placer({
          key: "test",
          location: "mosaic",
          type: "cat",
          name: "test",
          window: {
            title: "test",
          },
        });
      });
      act(() => {
        result.current.remover("test");
      });
      const state = select(store.getState(), "test");
      expect(state).toBeUndefined();
    });
  });
  describe("useSelectActiveMosaicTab", () => {
    it("should select the active mosaic tab", () => {
      const store = configureStore({
        reducer: combineReducers({
          [Layout.SLICE_NAME]: Layout.reducer,
          [Drift.SLICE_NAME]: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>
          <Modals.Provider>{children}</Modals.Provider>
        </Provider>
      );
      const { result } = renderHook(
        () => ({
          placer: Layout.usePlacer(),
          store: useStore(),
          activeTab: Layout.useSelectActiveMosaicTabState(),
        }),
        { wrapper },
      );

      // Initially there should be no active tab
      expect(result.current.activeTab).toEqual({
        blurred: false,
        layoutKey: null,
      });

      // Place a layout in the mosaic
      act(() => {
        result.current.placer({
          key: "test-tab",
          location: "mosaic",
          type: "cat",
          name: "Test Tab",
          window: {
            title: "test",
          },
        });
      });

      // Now the active tab should be the one we just placed
      expect(result.current.activeTab).toEqual({
        blurred: false,
        layoutKey: "test-tab",
      });
    });
    it("should return true for blurred if there is a modal open", () => {
      const store = configureStore({
        reducer: combineReducers({
          [Layout.SLICE_NAME]: Layout.reducer,
          [Drift.SLICE_NAME]: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>
          <Modals.Provider>{children}</Modals.Provider>
        </Provider>
      );
      const { result } = renderHook(
        () => ({
          placer: Layout.usePlacer(),
          store: useStore(),
          activeTab: Layout.useSelectActiveMosaicTabState(),
          modals: Modals.useStore("test"),
        }),
        { wrapper },
      );

      // Place a layout in the mosaic
      act(() => {
        result.current.placer({
          key: "test-tab",
          location: "mosaic",
          type: "cat",
          name: "Test Tab",
          window: {
            title: "test",
          },
        });
      });

      // Verify the tab is active
      expect(result.current.activeTab).toEqual({
        blurred: false,
        layoutKey: "test-tab",
      });

      // Open a modal in the independent modal store
      act(() => {
        result.current.modals.push(
          () => null,
          undefined,
          () => {},
        );
      });

      // The active tab is now blurred because a modal is open
      expect(result.current.activeTab).toEqual({
        blurred: true,
        layoutKey: "test-tab",
      });
      result.current.modals.clear();
    });
  });
});
