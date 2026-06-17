// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { act, type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { select } from "@/layout/selectors";
import { ConsoleTestProvider, createTestStore } from "@/testUtils";

describe("layout hooks", () => {
  describe("placing & removing", () => {
    it("should store a placed window or modal layout", () => {
      const store = createTestStore();
      const wrapper = ({ children }: PropsWithChildren) => (
        <ConsoleTestProvider store={store}>{children}</ConsoleTestProvider>
      );
      const { result } = renderHook(() => ({ placer: Layout.usePlacer() }), {
        wrapper,
      });
      act(() => {
        result.current.placer({
          key: "test",
          location: "modal",
          type: "cat",
          name: "test",
        });
      });
      const state = select(store.getState(), "test");
      expect(state).toBeDefined();
      expect(state?.key).toBe("test");
      expect(state?.type).toBe("cat");
      expect(state?.name).toBe("test");
    });

    // Document content (location "mosaic") routes into the active panel through
    // panel actions. With no active panel there is nowhere for it to land, so the
    // placer surfaces a warning instead of storing a layout nothing can render.
    it("should not store document content when there is no active panel", () => {
      const store = createTestStore();
      const wrapper = ({ children }: PropsWithChildren) => (
        <ConsoleTestProvider store={store}>{children}</ConsoleTestProvider>
      );
      const { result } = renderHook(() => ({ placer: Layout.usePlacer() }), {
        wrapper,
      });
      act(() => {
        result.current.placer({
          key: "test",
          location: "mosaic",
          type: "cat",
          name: "test",
        });
      });
      expect(select(store.getState(), "test")).toBeUndefined();
    });

    it("should remove a layout from the store", () => {
      const store = createTestStore();
      const wrapper = ({ children }: PropsWithChildren) => (
        <ConsoleTestProvider store={store}>{children}</ConsoleTestProvider>
      );
      const { result } = renderHook(
        () => ({ placer: Layout.usePlacer(), remover: Layout.useRemover() }),
        { wrapper },
      );
      act(() => {
        result.current.placer({
          key: "test",
          location: "modal",
          type: "cat",
          name: "test",
        });
      });
      act(() => {
        result.current.remover("test");
      });
      const state = select(store.getState(), "test");
      expect(state).toBeUndefined();
    });
  });
});
