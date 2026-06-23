// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  reducer,
  select,
  toggle,
  ZERO_SLICE_STATE,
} from "@/layered/session/theme/slice";

describe("Theme Slice", () => {
  it("should have the correct initial state", () => {
    expect(ZERO_SLICE_STATE.selected).toBe("synnaxLight");
  });

  it("should select a theme", () => {
    const state = reducer(ZERO_SLICE_STATE, select("synnaxDark"));
    expect(state.selected).toBe("synnaxDark");
  });

  it("should toggle between themes", () => {
    let state = ZERO_SLICE_STATE;
    state = reducer(state, toggle());
    expect(state.selected).toBe("synnaxDark");
    state = reducer(state, toggle());
    expect(state.selected).toBe("synnaxLight");
  });
});
