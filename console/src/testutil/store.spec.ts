// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice } from "@reduxjs/toolkit";
import { describe, expect, it, vi } from "vitest";

import { createMinimalStore } from "@/testutil";

const slice = createSlice({
  name: "counter",
  initialState: { count: 0 },
  reducers: {
    increment: (state) => {
      state.count += 1;
    },
  },
});

describe("createMinimalStore", () => {
  it("should not warn when its state checks run slowly", () => {
    const store = createMinimalStore({ reducer: slice.reducer });
    let now = 0;
    vi.spyOn(Date, "now").mockImplementation(() => (now += 40));
    const warn = vi.spyOn(console, "warn");
    store.dispatch(slice.actions.increment());
    expect(warn).not.toHaveBeenCalled();
  });

  it("should still run the immutability check", () => {
    const store = createMinimalStore({
      reducer: (state: { items: number[] } = { items: [] }) => state,
    });
    store.getState().items.push(1);
    expect(() => store.dispatch({ type: "any" })).toThrow(/mutation/);
  });

  it("should let the caller override a check", () => {
    const error = vi.spyOn(console, "error");
    const store = createMinimalStore({
      reducer: (state: object = {}) => state,
      middleware: (getDefault) => getDefault({ serializableCheck: false }),
    });
    store.dispatch({ type: "any", payload: new Map() });
    expect(error).not.toHaveBeenCalled();
  });
});
