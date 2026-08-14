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
  it("should default to following the system theme", () => {
    expect(Theme.ZERO_SLICE_STATE.mode).toBe("system");
  });

  it("should set the mode", () => {
    let state = Theme.reducer(Theme.ZERO_SLICE_STATE, Theme.set("dark"));
    expect(state.mode).toBe("dark");
    state = Theme.reducer(state, Theme.set("light"));
    expect(state.mode).toBe("light");
    state = Theme.reducer(state, Theme.set("system"));
    expect(state.mode).toBe("system");
  });
});
