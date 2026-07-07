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

import { Docs } from "@/session/docs";

const createStore = () =>
  configureStore({ reducer: { [Docs.SLICE_NAME]: Docs.reducer } });

const createWrapper =
  (store: ReturnType<typeof createStore>) =>
  ({ children }: PropsWithChildren): ReactElement =>
    <Provider store={store}>{children}</Provider>;

describe("docs selectors", () => {
  describe("useSelectLocation", () => {
    it("should return the zero location by default", () => {
      const store = createStore();
      const { result } = renderHook(() => Docs.useSelectLocation(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toEqual({ path: "", heading: "" });
    });

    it("should reflect a dispatched location change", () => {
      const store = createStore();
      const location: Docs.Location = { path: "/reference", heading: "api" };
      const { result } = renderHook(() => Docs.useSelectLocation(), {
        wrapper: createWrapper(store),
      });
      act(() => {
        store.dispatch(Docs.setLocation(location));
      });
      expect(result.current).toEqual(location);
    });
  });

  describe("useGetLocation", () => {
    it("should read the current location on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Docs.useGetLocation(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toEqual({ path: "", heading: "" });
      const first: Docs.Location = { path: "/guides/intro", heading: "setup" };
      act(() => {
        store.dispatch(Docs.setLocation(first));
      });
      expect(get()).toEqual(first);
      const second: Docs.Location = { path: "/reference", heading: "api" };
      act(() => {
        store.dispatch(Docs.setLocation(second));
      });
      expect(get()).toEqual(second);
    });
  });
});
