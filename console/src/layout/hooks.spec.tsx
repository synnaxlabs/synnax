// Copyright 2025 Synnax Labs, Inc.
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

import { Layout } from "@/layout";
import { select } from "@/layout/selectors";
import { reducer } from "@/layout/slice";

describe("layout hooks", () => {
  describe("placing & removing", () => {
    it("should place a layout within the store", () => {
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
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
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
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
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result } = renderHook(
        () => ({
          placer: Layout.usePlacer(),
          store: useStore(),
          activeTab: Layout.useSelectActiveMosaicTabKey(),
        }),
        { wrapper },
      );

      // Initially there should be no active tab
      expect(result.current.activeTab).toBeNull();

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
      expect(result.current.activeTab).toBe("test-tab");
    });
    it("should return null if there is a modal open", () => {
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result } = renderHook(
        () => ({
          placer: Layout.usePlacer(),
          store: useStore(),
          activeTab: Layout.useSelectActiveMosaicTabKey(),
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
      expect(result.current.activeTab).toBe("test-tab");

      // Place a modal
      act(() => {
        result.current.placer({
          key: "test-modal",
          location: "modal",
          type: "dog",
          name: "Test Modal",
          window: {
            title: "modal",
          },
        });
      });

      // Now the active tab should be null because a modal is open
      expect(result.current.activeTab).toBeNull();
    });
  });
});
