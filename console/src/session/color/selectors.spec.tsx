// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Color as PColor } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Color } from "@/session/color";

const createStore = () =>
  configureStore({ reducer: { [Color.SLICE_NAME]: Color.reducer } });

const createWrapper = (
  store: ReturnType<typeof createStore>,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const withFrequent: PColor.ContextState = {
  ...PColor.ZERO_CONTEXT_STATE,
  frequent: { "#FF0000": { lastUsed: 1, count: 2, relevance: 3 } },
};

describe("color selectors", () => {
  describe("useSelectContext", () => {
    it("should return the zero context by default", () => {
      const store = createStore();
      const { result } = renderHook(() => Color.useSelectContext(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toEqual(PColor.ZERO_CONTEXT_STATE);
    });

    it("should reflect a dispatched context change", () => {
      const store = createStore();
      const { result } = renderHook(() => Color.useSelectContext(), {
        wrapper: createWrapper(store),
      });
      act(() => {
        store.dispatch(Color.setContext(withFrequent));
      });
      expect(result.current).toEqual(withFrequent);
    });
  });

  describe("useGetContext", () => {
    it("should read the current context on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Color.useGetContext(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toEqual(PColor.ZERO_CONTEXT_STATE);
      act(() => {
        store.dispatch(Color.setContext(withFrequent));
      });
      expect(get()).toEqual(withFrequent);
    });
  });

  describe("PROVIDER_PROPS", () => {
    const renderProviderState = (store: ReturnType<typeof createStore>) =>
      renderHook(() => Color.PROVIDER_PROPS.useState?.(PColor.ZERO_CONTEXT_STATE), {
        wrapper: createWrapper(store),
      });

    it("exposes a useState hook reflecting the color context", () => {
      const store = createStore();
      const { result } = renderProviderState(store);
      expect(result.current?.[0]).toEqual(PColor.ZERO_CONTEXT_STATE);
      act(() => {
        store.dispatch(Color.setContext(withFrequent));
      });
      expect(result.current?.[0]).toEqual(withFrequent);
    });

    it("dispatches setContext when the setter is called", () => {
      const store = createStore();
      const { result } = renderProviderState(store);
      act(() => result.current?.[1](withFrequent));
      expect(store.getState()[Color.SLICE_NAME].context).toEqual(withFrequent);
    });
  });
});
