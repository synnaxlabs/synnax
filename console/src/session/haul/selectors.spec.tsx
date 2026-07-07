// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { type Haul as PHaul } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Haul } from "@/session/haul";

const createStore = () =>
  configureStore({ reducer: { [Haul.SLICE_NAME]: Haul.reducer } });

const createWrapper =
  (store: ReturnType<typeof createStore>) =>
  ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );

const dragging: PHaul.DraggingState = {
  source: { key: "src", type: "channel" },
  items: [{ key: "a", type: "channel" }],
};

describe("haul selectors", () => {
  describe("useSelectHauling", () => {
    it("should return the zero dragging state by default", () => {
      const store = createStore();
      const { result } = renderHook(() => Haul.useSelectHauling(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toEqual(Haul.ZERO_SLICE_STATE.state);
    });

    it("should reflect a dispatched haul", () => {
      const store = createStore();
      const { result } = renderHook(() => Haul.useSelectHauling(), {
        wrapper: createWrapper(store),
      });
      act(() => {
        store.dispatch(Haul.setHauled(dragging));
      });
      expect(result.current).toEqual(dragging);
    });
  });

  describe("useGetHauling", () => {
    it("should read the current dragging state on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Haul.useGetHauling(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toEqual(Haul.ZERO_SLICE_STATE.state);
      act(() => {
        store.dispatch(Haul.setHauled(dragging));
      });
      expect(get()).toEqual(dragging);
    });
  });

  describe("useProviderProps", () => {
    const renderProviderState = (store: ReturnType<typeof createStore>) =>
      renderHook(
        () => Haul.useProviderProps().useState?.(Haul.ZERO_SLICE_STATE.state),
        { wrapper: createWrapper(store) },
      );

    it("exposes a useState hook reflecting the dragging state", () => {
      const store = createStore();
      const { result } = renderProviderState(store);
      expect(result.current?.[0]).toEqual(Haul.ZERO_SLICE_STATE.state);
      act(() => {
        store.dispatch(Haul.setHauled(dragging));
      });
      expect(result.current?.[0]).toEqual(dragging);
    });

    it("dispatches setHauled when the setter is called", () => {
      const store = createStore();
      const { result } = renderProviderState(store);
      act(() => result.current?.[1](dragging));
      expect(store.getState()[Haul.SLICE_NAME].state).toEqual(dragging);
    });
  });
});
