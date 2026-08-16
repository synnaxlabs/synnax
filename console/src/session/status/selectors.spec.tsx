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
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Status } from "@/session/status";

const storeWith = (...actions: Status.Action[]) => {
  const store = configureStore({ reducer: { [Status.SLICE_NAME]: Status.reducer } });
  actions.forEach((action) => store.dispatch(action));
  return store;
};

const wrapperFor = (store: ReturnType<typeof storeWith>) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("status selectors", () => {
  describe("useSelectFavorites", () => {
    it("should track additions to the favorites list", () => {
      const store = storeWith(Status.addFavorites("a"));
      const { result } = renderHook(() => Status.useSelectFavorites(), {
        wrapper: wrapperFor(store),
      });
      expect(result.current).toEqual(["a"]);
      act(() => {
        store.dispatch(Status.addFavorites("b"));
      });
      expect(result.current).toEqual(["a", "b"]);
    });
  });

  describe("useSelectFavoriteSet", () => {
    it("should expose the favorites as a set", () => {
      const store = storeWith(Status.addFavorites(["a", "b"]));
      const { result } = renderHook(() => Status.useSelectFavoriteSet(), {
        wrapper: wrapperFor(store),
      });
      expect(result.current).toBeInstanceOf(Set);
      expect([...result.current]).toEqual(["a", "b"]);
    });
  });

  describe("useSelectIsFavorite", () => {
    it("should reflect toggling a key on and off", () => {
      const store = storeWith();
      const { result } = renderHook(() => Status.useSelectIsFavorite("a"), {
        wrapper: wrapperFor(store),
      });
      expect(result.current).toBe(false);
      act(() => {
        store.dispatch(Status.toggleFavorite("a"));
      });
      expect(result.current).toBe(true);
      act(() => {
        store.dispatch(Status.toggleFavorite("a"));
      });
      expect(result.current).toBe(false);
    });
  });
});
