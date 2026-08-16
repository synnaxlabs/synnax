// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Persist } from "@/session/persist";

describe("Persist slice", () => {
  it("should start with no swap in flight", () => {
    expect(Persist.ZERO_SLICE_STATE.swapping).toBe(false);
  });

  it("should mark swapping across a beginSwap/hydrate pair", () => {
    const swapping = Persist.reducer(Persist.ZERO_SLICE_STATE, Persist.beginSwap());
    expect(swapping.swapping).toBe(true);
    const hydrated = Persist.reducer(swapping, Persist.hydrate({}));
    expect(hydrated.swapping).toBe(false);
  });

  it("should clear swapping when a failed swap ends without hydrating", () => {
    const swapping = Persist.reducer(Persist.ZERO_SLICE_STATE, Persist.beginSwap());
    const ended = Persist.reducer(swapping, Persist.endSwap());
    expect(ended.swapping).toBe(false);
  });
});
