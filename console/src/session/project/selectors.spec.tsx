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

import { Project } from "@/session/project";

const selected = "00000000-0000-0000-0000-000000000001";
const other = "00000000-0000-0000-0000-000000000002";

const withSelected = {
  [Project.SLICE_NAME]: { ...Project.ZERO_SLICE_STATE, selected },
};
const withoutSelected = { [Project.SLICE_NAME]: Project.ZERO_SLICE_STATE };

const createStore = () =>
  configureStore({ reducer: { [Project.SLICE_NAME]: Project.reducer } });

const createWrapper =
  (store: ReturnType<typeof createStore>) =>
  ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );

describe("project selectors", () => {
  describe("raw selectors", () => {
    it("should return the selected project from both selectors", () => {
      expect(Project.selectSelected(withSelected)).toEqual(selected);
      expect(Project.selectOptionalSelected(withSelected)).toEqual(selected);
    });

    it("should return undefined from the optional selector when unset", () => {
      expect(Project.selectOptionalSelected(withoutSelected)).toBeUndefined();
    });

    it("should throw from the required selector when unset", () => {
      expect(() => Project.selectSelected(withoutSelected)).toThrow();
    });
  });

  describe("getters", () => {
    it("should read the current selected project on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Project.useGetSelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      act(() => {
        store.dispatch(Project.select(selected));
      });
      expect(get()).toEqual(selected);
      act(() => {
        store.dispatch(Project.select(other));
      });
      expect(get()).toEqual(other);
    });

    it("should read the optional selected project without throwing when unset", () => {
      const store = createStore();
      const { result } = renderHook(() => Project.useGetOptionalSelected(), {
        wrapper: createWrapper(store),
      });
      expect(result.current()).toBeUndefined();
      act(() => {
        store.dispatch(Project.select(selected));
      });
      expect(result.current()).toEqual(selected);
    });

    it("should report whether any project is selected on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Project.useGetIsAnySelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        store.dispatch(Project.select(selected));
      });
      expect(get()).toBe(true);
    });
  });
});
