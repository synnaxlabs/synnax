// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Theme } from "@/session/theme";

describe("Theme Slice", () => {
  it("should have the correct initial state", () => {
    expect(Theme.ZERO_SLICE_STATE.selected).toBe("synnaxLight");
  });

  it("should select a theme", () => {
    const state = Theme.reducer(Theme.ZERO_SLICE_STATE, Theme.select("synnaxDark"));
    expect(state.selected).toBe("synnaxDark");
  });

  it("should toggle between themes", () => {
    let state = Theme.ZERO_SLICE_STATE;
    state = Theme.reducer(state, Theme.toggle());
    expect(state.selected).toBe("synnaxDark");
    state = Theme.reducer(state, Theme.toggle());
    expect(state.selected).toBe("synnaxLight");
  });
});
