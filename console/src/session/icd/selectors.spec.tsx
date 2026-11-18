// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { ICD } from "@/session/icd";
import { createSliceStore, inWindow } from "@/session/window/testutil";

const createWrapper = (slice: ICD.SliceState): FC<PropsWithChildren> => {
  const store = createSliceStore({
    name: ICD.SLICE_NAME,
    reducer: ICD.reducer,
    preloadedState: slice,
    middleware: ICD.MIDDLEWARE,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("ICD selector hooks", () => {
  it("should read the selected messages of an ICD", () => {
    const wrapper = createWrapper({
      windows: inWindow({ "icd-1": { selected: ["alarm"] } }),
    });
    const { result } = renderHook(() => ICD.useSelectSelected("icd-1"), { wrapper });
    expect(result.current).toEqual(["alarm"]);
  });

  it("should select nothing for an ICD with no state", () => {
    const wrapper = createWrapper(ICD.ZERO_SLICE_STATE);
    const { result } = renderHook(() => ICD.useSelectSelected("icd-1"), { wrapper });
    expect(result.current).toEqual([]);
  });
});
