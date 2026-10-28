// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { type Haul as PHaul } from "@synnaxlabs/lyra/haul";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Haul } from "@/session/haul";

const createStore = () =>
  configureStore({ reducer: { [Haul.SLICE_NAME]: Haul.reducer } });

const createWrapper = (
  store: ReturnType<typeof createStore>,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const dragging: PHaul.DraggingState = {
  source: { key: "src", type: "channel" },
  items: [{ key: "a", type: "channel" }],
};

describe("haul selectors", () => {
  describe("useProviderProps", () => {
    const renderProps = (store: ReturnType<typeof createStore>) =>
      renderHook(() => Haul.useProviderProps(), { wrapper: createWrapper(store) });

    it("reflects the dragging state", () => {
      const store = createStore();
      const { result } = renderProps(store);
      expect(result.current.value).toEqual(Haul.ZERO_SLICE_STATE.state);
      act(() => {
        store.dispatch(Haul.setHauled(dragging));
      });
      expect(result.current.value).toEqual(dragging);
    });

    it("dispatches setHauled on change", () => {
      const store = createStore();
      const { result } = renderProps(store);
      act(() => result.current.onChange?.(dragging));
      expect(store.getState()[Haul.SLICE_NAME].state).toEqual(dragging);
    });
  });
});
