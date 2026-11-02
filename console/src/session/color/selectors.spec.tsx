// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { Color as PColor } from "@synnaxlabs/lyra/color";
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
  describe("useProviderProps", () => {
    const renderProps = (store: ReturnType<typeof createStore>) =>
      renderHook(() => Color.useProviderProps(), { wrapper: createWrapper(store) });

    it("reflects the color context", () => {
      const store = createStore();
      const { result } = renderProps(store);
      expect(result.current.value).toEqual(PColor.ZERO_CONTEXT_STATE);
      act(() => {
        store.dispatch(Color.setContext(withFrequent));
      });
      expect(result.current.value).toEqual(withFrequent);
    });

    it("dispatches setContext on change", () => {
      const store = createStore();
      const { result } = renderProps(store);
      act(() => result.current.onChange?.(withFrequent));
      expect(store.getState()[Color.SLICE_NAME].context).toEqual(withFrequent);
    });
  });
});
