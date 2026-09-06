// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Link } from "@/session/link";

const createStore = () =>
  configureStore({ reducer: { [Link.SLICE_NAME]: Link.reducer } });

const createWrapper = (
  store: ReturnType<typeof createStore>,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("link selectors", () => {
  describe("useSelectAwaitingProject", () => {
    it("should be false by default", () => {
      const store = createStore();
      const { result } = renderHook(() => Link.useSelectAwaitingProject(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toBe(false);
    });

    it("should track the project wait across begin and end", () => {
      const store = createStore();
      const { result } = renderHook(() => Link.useSelectAwaitingProject(), {
        wrapper: createWrapper(store),
      });
      act(() => {
        store.dispatch(Link.beginProjectWait());
      });
      expect(result.current).toBe(true);
      act(() => {
        store.dispatch(Link.endProjectWait());
      });
      expect(result.current).toBe(false);
    });
  });
});
