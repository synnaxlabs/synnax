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
import { Icon } from "@synnaxlabs/pluto";
import { renderHook } from "@testing-library/react";
import { act, type PropsWithChildren } from "react";
import { Provider, useStore } from "react-redux";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { select } from "@/layout/selectors";
import { reducer } from "@/layout/slice";
import { type NavDrawerItem, useNavDrawer } from "@/layout/useNavDrawer";

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
          activeTab: Layout.useSelectActiveMosaicTabState(),
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
      expect(result.current.activeTab).toEqual({
        blurred: true,
        layoutKey: "test-tab",
      });
    });
  });

  describe("useNavDrawer", () => {
    const mockItems: NavDrawerItem[] = [
      {
        key: "channel",
        icon: <Icon.Channel />,
        tooltip: "Channel",
        trigger: ["H"],
        content: <div>Channel</div>,
      },
      {
        key: "range",
        icon: <Icon.Range />,
        tooltip: "Ranges",
        trigger: ["R"],
        content: <div>Ranges</div>,
      },
    ];
    const bottomItems: NavDrawerItem[] = [
      {
        key: "visualization",
        icon: <Icon.Visualize />,
        tooltip: "Visualize",
        trigger: ["V"],
        content: <div>Visualize</div>,
      },
    ];

    it("should initialize with default values", () => {
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result } = renderHook(() => useNavDrawer("left", mockItems), { wrapper });

      expect(result.current.activeItem).toBeUndefined();
      expect(result.current.menuItems).toHaveLength(mockItems.length);
      expect(result.current.hover).toBe(false);
    });

    it("should handle item selection and collapsing", () => {
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result } = renderHook(() => useNavDrawer("left", mockItems), { wrapper });

      act(() => {
        result.current.onSelect("channel");
      });
      expect(result.current.menuItems).toContain(mockItems[0]);

      act(() => {
        result.current.onCollapse();
      });
      expect(result.current.activeItem).toBeUndefined();
    });

    it("should handle hover interactions", () => {
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result } = renderHook(() => useNavDrawer("left", mockItems), { wrapper });

      act(() => {
        result.current.onStartHover("channel");
      });
      expect(result.current.hover).toBe(true);
      expect(result.current.activeItem).toEqual(mockItems[0]);

      act(() => {
        result.current.onStartHover("range");
      });
      expect(result.current.hover).toBe(true);
      expect(result.current.activeItem?.key).toBe("range");

      act(() => {
        result.current.onStopHover();
      });
      expect(result.current.hover).toBe(false);
      expect(result.current.activeItem).toBeUndefined();
    });

    it("should handle resizing with debounce", async () => {
      vi.useFakeTimers();
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result } = renderHook(() => useNavDrawer("left", mockItems), { wrapper });

      act(() => {
        result.current.onResize(200);
      });

      act(() => {
        vi.advanceTimersByTime(150);
      });

      vi.useRealTimers();
    });

    it("should handle multiple drawer locations", () => {
      const store = configureStore({
        reducer: combineReducers({
          layout: reducer,
          drift: Drift.reducer,
        }),
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Provider store={store}>{children}</Provider>
      );
      const { result: leftDrawer } = renderHook(() => useNavDrawer("left", mockItems), {
        wrapper,
      });
      const { result: bottomDrawer } = renderHook(
        () => useNavDrawer("bottom", bottomItems),
        {
          wrapper,
        },
      );

      act(() => {
        leftDrawer.current.onSelect("channel");
        bottomDrawer.current.onSelect("visualization");
      });

      expect(leftDrawer.current.activeItem?.key).toBe("channel");
      expect(bottomDrawer.current.activeItem?.key).toBe("visualization");
    });
  });
});
