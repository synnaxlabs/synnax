// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/platform/layout";
import { Session } from "@/session";
import { Modals } from "@/session/modals";
import { renderHookWithConsole } from "@/testutil";

describe("layout hooks", () => {
  describe("placing & removing", () => {
    it("should place a layout within the store", async () => {
      const { result, store } = await renderHookWithConsole(() => ({
        placer: Layout.usePlacer(),
      }));
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
      const state = Session.Layout.select(store.getState(), "test");
      expect(state).toBeDefined();
      expect(state?.key).toBe("test");
      expect(state?.location).toBe("mosaic");
      expect(state?.type).toBe("cat");
      expect(state?.name).toBe("test");
    });

    it("should remove a layout from the store", async () => {
      const { result, store } = await renderHookWithConsole(() => ({
        placer: Layout.usePlacer(),
        remover: Layout.useRemover(),
      }));
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
      const state = Session.Layout.select(store.getState(), "test");
      expect(state).toBeUndefined();
    });
  });
  describe("useSelectActiveMosaicTab", () => {
    it("should select the active mosaic tab", async () => {
      const { result } = await renderHookWithConsole(() => ({
        placer: Layout.usePlacer(),
        activeTab: Session.Layout.useSelectActiveMosaicTabState(),
      }));

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
    it("should return true for blurred if there is a modal open", async () => {
      const { result } = await renderHookWithConsole(() => ({
        placer: Layout.usePlacer(),
        activeTab: Session.Layout.useSelectActiveMosaicTabState(),
        modals: Modals.useStore("test"),
      }));

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

  describe("useOpenInNewWindow", () => {
    it("should create a mosaic window and move the layout's tab into it", async () => {
      const { result, store } = await renderHookWithConsole(() => ({
        placer: Layout.usePlacer(),
        openInNewWindow: Layout.useOpenInNewWindow(),
      }));

      act(() => {
        result.current.placer({
          key: "plot-1",
          location: "mosaic",
          type: "lineplot",
          name: "Plot 1",
        });
      });
      expect(Session.Layout.select(store.getState(), "plot-1")?.windowKey).toEqual(
        Drift.MAIN_WINDOW,
      );

      act(() => {
        result.current.openInNewWindow("plot-1");
      });

      const layouts = store.getState()[Session.Layout.SLICE_NAME].layouts;
      const newWindow = Object.values(layouts).find(
        (l) => l.type === Session.Layout.MOSAIC_WINDOW_TYPE,
      );
      expect(newWindow).toBeDefined();
      expect(Session.Layout.select(store.getState(), "plot-1")?.windowKey).toEqual(
        newWindow?.key,
      );
    });
  });
});
