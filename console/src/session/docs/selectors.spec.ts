// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Docs } from "@/session/docs";

describe("docs slice", () => {
  describe("setLocation", () => {
    it("should replace the current location", () => {
      const location: Docs.Location = { path: "/guides/intro", heading: "setup" };
      const state = Docs.reducer(Docs.ZERO_SLICE_STATE, Docs.setLocation(location));
      expect(state.location).toEqual(location);
    });
  });

  describe("selectSliceState", () => {
    it("should return the docs slice state", () => {
      const store: Docs.StoreState = { [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE };
      expect(Docs.selectSliceState(store)).toBe(Docs.ZERO_SLICE_STATE);
    });
  });

  describe("selectLocation", () => {
    it("should return the zero location by default", () => {
      const store: Docs.StoreState = { [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE };
      expect(Docs.selectLocation(store)).toEqual({ path: "", heading: "" });
    });

    it("should reflect a dispatched location change", () => {
      const location: Docs.Location = { path: "/reference", heading: "api" };
      const sliceState = Docs.reducer(
        Docs.ZERO_SLICE_STATE,
        Docs.setLocation(location),
      );
      const store: Docs.StoreState = { [Docs.SLICE_NAME]: sliceState };
      expect(Docs.selectLocation(store)).toEqual(location);
    });
  });
});
