// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

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

  describe("filterFavoritesToKeys", () => {
    it("should keep only favorites present in the payload", () => {
      let state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b", "c"]),
      );
      state = Status.reducer(state, Status.filterFavoritesToKeys(["b", "c", "d"]));
      expect(state.favorites).toEqual(["b", "c"]);
    });

    it("should clear favorites when the payload matches none", () => {
      let state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b"]),
      );
      state = Status.reducer(state, Status.filterFavoritesToKeys([]));
      expect(state.favorites).toEqual([]);
    });

    it("should accept a single key", () => {
      let state = Status.reducer(
        Status.ZERO_SLICE_STATE,
        Status.addFavorites(["a", "b"]),
      );
      state = Status.reducer(state, Status.filterFavoritesToKeys("a"));
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

  describe("migrateSlice", () => {
    it("should backfill missing fields from the zero state", () => {
      const migrated = Status.migrateSlice({ favorites: ["a"] });
      expect(migrated).toEqual({ version: 0, favorites: ["a"] });
    });

    it("should preserve existing fields", () => {
      const existing: Status.SliceState = { version: 0, favorites: ["x", "y"] };
      expect(Status.migrateSlice(existing)).toEqual(existing);
    });
  });
});
