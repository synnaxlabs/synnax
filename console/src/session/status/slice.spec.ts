// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Status } from "@/session/status";

describe("status slice", () => {
  describe("addFavorites", () => {
    it("should append a single favorite key", () => {
      const state = Status.reducer(Status.ZERO_SLICE_STATE, Status.addFavorites("a"));
      expect(state.favorites).toEqual(["a"]);
    });

    it("should append multiple favorite keys", () => {
      const state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b"]),
      );
      expect(state.favorites).toEqual(["a", "b"]);
    });

    it("should not add duplicates that already exist", () => {
      let state = Status.reducer(Status.ZERO_SLICE_STATE, Status.addFavorites("a"));
      state = Status.reducer(state, Status.addFavorites(["a", "b", "a"]));
      expect(state.favorites).toEqual(["a", "b"]);
    });

    it("should dedupe within a single payload", () => {
      const state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "a", "b"]),
      );
      expect(state.favorites).toEqual(["a", "b"]);
    });
  });

  describe("removeFavorites", () => {
    it("should remove a single favorite key", () => {
      let state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b", "c"]),
      );
      state = Status.reducer(state, Status.removeFavorites("b"));
      expect(state.favorites).toEqual(["a", "c"]);
    });

    it("should remove multiple favorite keys", () => {
      let state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b", "c"]),
      );
      state = Status.reducer(state, Status.removeFavorites(["a", "c"]));
      expect(state.favorites).toEqual(["b"]);
    });

    it("should be a no-op when given an empty key array", () => {
      const initial = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b"]),
      );
      const state = Status.reducer(initial, Status.removeFavorites([]));
      expect(state.favorites).toEqual(["a", "b"]);
      expect(state).toBe(initial);
    });

    it("should ignore keys that are not favorites", () => {
      let state = Status.reducer(Status.ZERO_SLICE_STATE, Status.addFavorites(["a"]));
      state = Status.reducer(state, Status.removeFavorites("z"));
      expect(state.favorites).toEqual(["a"]);
    });
  });

  describe("toggleFavorite", () => {
    it("should add a favorite that is not present", () => {
      const state = Status.reducer(Status.ZERO_SLICE_STATE, Status.toggleFavorite("a"));
      expect(state.favorites).toEqual(["a"]);
    });

    it("should remove a favorite that is already present", () => {
      let state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b"]),
      );
      state = Status.reducer(state, Status.toggleFavorite("a"));
      expect(state.favorites).toEqual(["b"]);
    });

    it("should round-trip back to the original set after two toggles", () => {
      let state = Status.reducer(Status.ZERO_SLICE_STATE, Status.addFavorites(["b"]));
      state = Status.reducer(state, Status.toggleFavorite("a"));
      state = Status.reducer(state, Status.toggleFavorite("a"));
      expect(state.favorites).toEqual(["b"]);
    });
  });

  describe("sliceStateZ", () => {
    it("should move a favorite to the key the Core gives its owner's status", () => {
      const state = Status.sliceStateZ.parse({ version: 0, favorites: ["rack:1"] });
      expect(state.favorites).toEqual(["5decc7ad-b3cc-5bde-8216-0d1df8f0ac55"]);
    });

    it.each(["task:0f6a4f7e-3f1b-4a51-9b0e-2b8f3c6d1e2a", "device:dev-1", "channel:5"])(
      "should move the favorite %s to a UUID",
      (key) => {
        const state = Status.sliceStateZ.parse({ version: 0, favorites: [key] });
        expect(z.uuid().safeParse(state.favorites[0]).success).toBe(true);
      },
    );

    it.each([
      "pressure-alarm",
      "5decc7ad-b3cc-5bde-8216-0d1df8f0ac55",
      "label:0f6a4f7e-3f1b-4a51-9b0e-2b8f3c6d1e2a",
    ])("should keep the favorite %s", (key) => {
      const state = Status.sliceStateZ.parse({ version: 0, favorites: [key] });
      expect(state.favorites).toEqual([key]);
    });

    it("should save loaded state as version 1", () => {
      const state = Status.sliceStateZ.parse({ version: 0, favorites: ["rack:1"] });
      expect(state.version).toBe(1);
    });

    it("should keep every favorite of version 1 state", () => {
      const state = Status.sliceStateZ.parse({
        version: 1,
        favorites: ["device:pump-1"],
      });
      expect(state.favorites).toEqual(["device:pump-1"]);
    });
  });
});
