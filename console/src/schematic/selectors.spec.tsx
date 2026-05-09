// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { selectIsRemoteCreated, useSelectIsRemoteCreated } from "@/schematic/selectors";
import {
  reducer,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/slice";

const buildState = (overrides: Partial<State>): StoreState => {
  const key = overrides.key ?? "schematic-1";
  const slice: SliceState = {
    ...ZERO_SLICE_STATE,
    schematics: {
      ...ZERO_SLICE_STATE.schematics,
      [key]: { ...ZERO_STATE, ...overrides, key },
    },
  };
  return { [SLICE_NAME]: slice };
};

const wrapperFor = (state: StoreState) => {
  const store = configureStore({
    reducer: { [SLICE_NAME]: reducer },
    preloadedState: state,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("schematic selectors", () => {
  describe("selectIsRemoteCreated", () => {
    it("should return true when the schematic has been remotely created", () => {
      const state = buildState({ key: "schematic-1", remoteCreated: true });
      expect(selectIsRemoteCreated(state, "schematic-1")).toBe(true);
    });

    it("should return false when the schematic has not been remotely created", () => {
      const state = buildState({ key: "schematic-1", remoteCreated: false });
      expect(selectIsRemoteCreated(state, "schematic-1")).toBe(false);
    });

    it("should return undefined when the schematic is not in the slice", () => {
      const state: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };
      expect(selectIsRemoteCreated(state, "missing")).toBeUndefined();
    });
  });

  describe("useSelectIsRemoteCreated", () => {
    it("should expose the remoteCreated flag from the slice", () => {
      const state = buildState({ key: "schematic-1", remoteCreated: true });
      const { result } = renderHook(() => useSelectIsRemoteCreated("schematic-1"), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBe(true);
    });

    it("should return undefined for a missing schematic key", () => {
      const state: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };
      const { result } = renderHook(() => useSelectIsRemoteCreated("missing"), {
        wrapper: wrapperFor(state),
      });
      expect(result.current).toBeUndefined();
    });
  });
});
