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
      v6.ZERO_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect(migrated.version).toBe(ZERO_STATE.version);
        expect(migrated.activeToolbarTab).toBe(ZERO_STATE.activeToolbarTab);
        expect(migrated.selected).toEqual([]);
        expect(migrated.legend).toBeDefined();
        expect(migrated.viewport).toBeDefined();
      });
    });

    it("should park v5 graph state into pendingUpload when migrating to v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        nodes: [{ key: "n1", position: { x: 0, y: 0 } } as v0.Node],
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            sourceHandle: "1",
            targetHandle: "2",
          } as v0.Edge,
        ],
        props: { n1: { key: "valve", color: "#ff0000" } as v0.NodeProps },
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload).toBeDefined();
      expect(migrated.pendingUpload?.nodes).toHaveLength(1);
      expect(migrated.pendingUpload?.edges).toHaveLength(1);
      expect(migrated.pendingUpload?.props).toEqual({
        n1: { key: "valve", color: "#ff0000" },
      });
    });

    it("should add an empty selected array when migrating to v6", () => {
      const migrated = migrateState(v5.ZERO_STATE);
      expect(migrated.selected).toEqual([]);
    });

    it("should preserve legend visibility and position when migrating v5 → v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        legend: {
          ...v5.ZERO_STATE.legend,
          visible: true,
          position: { x: 123, y: 456, units: { x: "px", y: "px" } },
        },
      };
      const migrated = migrateState(populated);
      expect(migrated.legend.visible).toBe(true);
      expect(migrated.legend.position).toEqual({
        x: 123,
        y: 456,
        units: { x: "px", y: "px" },
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
      v6.ZERO_SLICE_STATE,
    ];
    STATES.forEach((state) => {
      it(`should migrate slice from ${state.version} to latest`, () => {
        const migrated = migrateSlice(state);
        expect(migrated).toEqual(ZERO_SLICE_STATE);
      });
    });
  });
});
