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
  pinLeft,
  reducer,
  resizeBottom,
  resizeLeft,
  selectBottom,
  selectLeft,
  setBottomVisible,
  startBottomHover,
  startLeftHover,
  stopBottomHover,
  stopLeftHover,
  toggleBottom,
  toggleLeft,
  ZERO_SLICE_STATE,
} from "@/nav/slice";

describe("nav slice", () => {
  describe("zero state", () => {
    it("should start with no left selection and a hidden bottom", () => {
      expect(ZERO_SLICE_STATE.left.selected).toBeUndefined();
      expect(ZERO_SLICE_STATE.left.hover).toBe(false);
      expect(ZERO_SLICE_STATE.bottom.visible).toBe(false);
      expect(ZERO_SLICE_STATE.bottom.hover).toBe(false);
    });
  });

  describe("left drawer", () => {
    it("should select an item on selectLeft", () => {
      const state = reducer(ZERO_SLICE_STATE, selectLeft("channel"));
      expect(state.left.selected).toBe("channel");
      expect(state.left.hover).toBe(false);
    });

    it("should deselect a pinned item when selectLeft is called with the same key", () => {
      const pinned = reducer(ZERO_SLICE_STATE, selectLeft("channel"));
      const state = reducer(pinned, selectLeft("channel"));
      expect(state.left.selected).toBeUndefined();
    });

    it("should switch the pinned item when selectLeft targets a different key", () => {
      const pinned = reducer(ZERO_SLICE_STATE, selectLeft("channel"));
      const state = reducer(pinned, selectLeft("range"));
      expect(state.left.selected).toBe("range");
    });

    it("should pin an item and clear hover on pinLeft", () => {
      const peeking = reducer(ZERO_SLICE_STATE, startLeftHover("range"));
      const state = reducer(peeking, pinLeft("range"));
      expect(state.left.selected).toBe("range");
      expect(state.left.hover).toBe(false);
    });

    it("should peek an item on startLeftHover", () => {
      const state = reducer(ZERO_SLICE_STATE, startLeftHover("range"));
      expect(state.left.selected).toBe("range");
      expect(state.left.hover).toBe(true);
    });

    it("should not peek over a pinned item", () => {
      const pinned = reducer(ZERO_SLICE_STATE, selectLeft("channel"));
      const state = reducer(pinned, startLeftHover("range"));
      expect(state.left.selected).toBe("channel");
      expect(state.left.hover).toBe(false);
    });

    it("should clear the peek on stopLeftHover", () => {
      const peeking = reducer(ZERO_SLICE_STATE, startLeftHover("range"));
      const state = reducer(peeking, stopLeftHover());
      expect(state.left.selected).toBeUndefined();
      expect(state.left.hover).toBe(false);
    });

    it("should not clear a pinned item on stopLeftHover", () => {
      const pinned = reducer(ZERO_SLICE_STATE, selectLeft("channel"));
      const state = reducer(pinned, stopLeftHover());
      expect(state.left.selected).toBe("channel");
    });

    it("should toggle a hidden drawer into a peek on toggleLeft", () => {
      const state = reducer(ZERO_SLICE_STATE, toggleLeft("range"));
      expect(state.left.selected).toBe("range");
      expect(state.left.hover).toBe(true);
    });

    it("should close a pinned drawer on toggleLeft with the same key", () => {
      const pinned = reducer(ZERO_SLICE_STATE, selectLeft("range"));
      const state = reducer(pinned, toggleLeft("range"));
      expect(state.left.selected).toBeUndefined();
    });

    it("should store the resized size", () => {
      const state = reducer(ZERO_SLICE_STATE, resizeLeft(220));
      expect(state.left.size).toBe(220);
    });
  });

  describe("bottom drawer", () => {
    it("should show the drawer on selectBottom", () => {
      const state = reducer(ZERO_SLICE_STATE, selectBottom());
      expect(state.bottom.visible).toBe(true);
      expect(state.bottom.hover).toBe(false);
    });

    it("should hide a pinned drawer on selectBottom", () => {
      const shown = reducer(ZERO_SLICE_STATE, selectBottom());
      const state = reducer(shown, selectBottom());
      expect(state.bottom.visible).toBe(false);
    });

    it("should set visibility explicitly on setBottomVisible", () => {
      const state = reducer(ZERO_SLICE_STATE, setBottomVisible(true));
      expect(state.bottom.visible).toBe(true);
      expect(state.bottom.hover).toBe(false);
    });

    it("should peek the drawer on startBottomHover", () => {
      const state = reducer(ZERO_SLICE_STATE, startBottomHover());
      expect(state.bottom.visible).toBe(true);
      expect(state.bottom.hover).toBe(true);
    });

    it("should clear the peek on stopBottomHover", () => {
      const peeking = reducer(ZERO_SLICE_STATE, startBottomHover());
      const state = reducer(peeking, stopBottomHover());
      expect(state.bottom.visible).toBe(false);
      expect(state.bottom.hover).toBe(false);
    });

    it("should peek a hidden drawer on toggleBottom", () => {
      const state = reducer(ZERO_SLICE_STATE, toggleBottom());
      expect(state.bottom.visible).toBe(true);
      expect(state.bottom.hover).toBe(true);
    });

    it("should pin a peeking drawer on toggleBottom", () => {
      const peeking = reducer(ZERO_SLICE_STATE, startBottomHover());
      const state = reducer(peeking, toggleBottom());
      expect(state.bottom.visible).toBe(true);
      expect(state.bottom.hover).toBe(false);
    });

    it("should hide a pinned drawer on toggleBottom", () => {
      const pinned = reducer(ZERO_SLICE_STATE, setBottomVisible(true));
      const state = reducer(pinned, toggleBottom());
      expect(state.bottom.visible).toBe(false);
    });

    it("should store the resized size", () => {
      const state = reducer(ZERO_SLICE_STATE, resizeBottom(180));
      expect(state.bottom.size).toBe(180);
    });
  });
});
