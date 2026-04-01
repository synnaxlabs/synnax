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
  migrateSlice,
  migrateState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/types";
import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import * as v2 from "@/schematic/types/v2";
import * as v3 from "@/schematic/types/v3";
import * as v4 from "@/schematic/types/v4";
import * as v5 from "@/schematic/types/v5";
import * as v6 from "@/schematic/types/v6";

describe("migrations", () => {
  describe("state", () => {
    const STATES = [
      v0.ZERO_STATE,
      v1.ZERO_STATE,
      v2.ZERO_STATE,
      v3.ZERO_STATE,
      v4.ZERO_STATE,
      v5.ZERO_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect(migrated.version).toEqual(v6.VERSION);
        expect(migrated.selected).toEqual([]);
        expect(migrated.control).toEqual("released");
        expect(migrated.editable).toEqual(true);
        expect(migrated.fitViewOnResize).toEqual(false);
        expect(migrated.activeToolbarTab).toEqual("symbols");
        expect(migrated.selectedSymbolGroup).toEqual("general");
        expect(migrated).not.toHaveProperty("nodes");
        expect(migrated).not.toHaveProperty("edges");
        expect(migrated).not.toHaveProperty("props");
        expect(migrated).not.toHaveProperty("snapshot");
        expect(migrated).not.toHaveProperty("remoteCreated");
        expect(migrated).not.toHaveProperty("authority");
      });
    });

    it("should pass through v6 state unchanged", () => {
      const migrated = migrateState(v6.ZERO_STATE);
      expect(migrated).toEqual(ZERO_STATE);
    });

    it("should extract UI fields from a populated v5 state", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        editable: false,
        fitViewOnResize: true,
        control: "acquired",
        legend: {
          visible: true,
          position: { x: 100, y: 200, units: { x: "px", y: "px" } },
          colors: { chan1: "#ff0000" },
        },
        toolbar: { activeTab: "properties", selectedSymbolGroup: "valves" },
        viewport: { position: { x: 10, y: 20 }, zoom: 2 },
      };
      const migrated = migrateState(populated);
      expect(migrated).toEqual({
        version: v6.VERSION,
        selected: [],
        control: "acquired",
        legend: {
          visible: true,
          position: { x: 100, y: 200, units: { x: "px", y: "px" } },
        },
        activeToolbarTab: "properties",
        selectedSymbolGroup: "valves",
        editable: false,
        fitViewOnResize: true,
        viewport: { position: { x: 10, y: 20 }, zoom: 2 },
      });
    });
  });

  describe("slice", () => {
    const STATES = [
      v0.ZERO_SLICE_STATE,
      v1.ZERO_SLICE_STATE,
      v2.ZERO_SLICE_STATE,
      v3.ZERO_SLICE_STATE,
      v4.ZERO_SLICE_STATE,
      v5.ZERO_SLICE_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });

    it("should pass through v6 slice unchanged", () => {
      const migrated = migrateSlice(v6.ZERO_SLICE_STATE);
      expect(migrated).toEqual(ZERO_SLICE_STATE);
    });
  });
});
