// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { maybeRename, reducer, setActive } from "@/project/slice";
import { ZERO_SLICE_STATE } from "@/project/types";

describe("project slice", () => {
  const active = { key: "00000000-0000-0000-0000-000000000001", name: "Ops" };

  describe("setActive", () => {
    it("should set the active project", () => {
      expect(reducer(ZERO_SLICE_STATE, setActive(active)).active).toEqual(active);
    });

    it("should clear the active project when set to null", () => {
      expect(
        reducer({ ...ZERO_SLICE_STATE, active }, setActive(null)).active,
      ).toBeNull();
    });
  });

  describe("maybeRename", () => {
    it("should rename the active project when the key matches", () => {
      const next = reducer(
        { ...ZERO_SLICE_STATE, active },
        maybeRename({ key: active.key, name: "Renamed" }),
      );
      expect(next.active).toEqual({ key: active.key, name: "Renamed" });
    });

    it("should leave the active project unchanged when the key does not match", () => {
      const next = reducer(
        { ...ZERO_SLICE_STATE, active },
        maybeRename({ key: "00000000-0000-0000-0000-000000000099", name: "Renamed" }),
      );
      expect(next.active).toEqual(active);
    });

    it("should be a no-op when there is no active project", () => {
      const next = reducer(
        ZERO_SLICE_STATE,
        maybeRename({ key: active.key, name: "Renamed" }),
      );
      expect(next.active).toBeNull();
    });
  });
});
