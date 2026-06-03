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
  selectActiveToolbarTab,
  selectEditable,
  selectFitViewOnResize,
  selectLegendVisible,
  selectSelectedSymbolGroup,
  selectToolbar,
  selectViewport,
} from "@/schematic/selectors";
import {
  SLICE_NAME,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/slice";

describe("schematic selectors", () => {
  describe("missing slice entry", () => {
    const state: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };

    it("should return undefined from selectToolbar without throwing", () => {
      expect(selectToolbar(state, "absent")).toBeUndefined();
    });

    it("should return undefined from selectActiveToolbarTab without throwing", () => {
      expect(selectActiveToolbarTab(state, "absent")).toBeUndefined();
    });

    it("should default selectEditable to the zero value", () => {
      expect(selectEditable(state, "absent")).toBe(ZERO_STATE.editable);
    });

    it("should default selectSelectedSymbolGroup to the zero value", () => {
      expect(selectSelectedSymbolGroup(state, "absent")).toBe(
        ZERO_STATE.toolbar.selectedSymbolGroup,
      );
    });

    it("should default selectLegendVisible to the zero value", () => {
      expect(selectLegendVisible(state, "absent")).toBe(ZERO_STATE.legend.visible);
    });

    it("should default selectFitViewOnResize to the zero value", () => {
      expect(selectFitViewOnResize(state, "absent")).toBe(ZERO_STATE.fitViewOnResize);
    });

    it("should default selectViewport to the zero value", () => {
      expect(selectViewport(state, "absent")).toBe(ZERO_STATE.viewport);
    });
  });
});
