// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
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
        expect({ ...migrated, key: expect.anything() }).toEqual(ZERO_STATE);
      });
    });

    it("should rename nodePropsZ.key to .variant when migrating v5 → v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        props: {
          n1: { key: "valve", color: "#ff0000" } as v0.NodeProps,
          n2: { key: "tank" } as v0.NodeProps,
        },
      };
      const migrated = migrateState(populated);
      expect(migrated.props.n1).toMatchObject({ variant: "valve", color: "#ff0000" });
      expect(migrated.props.n2).toMatchObject({ variant: "tank" });
      expect(migrated.props.n1).not.toHaveProperty("key");
    });

    it("should reshape edge endpoints to Handle objects when migrating v5 → v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            sourceHandle: "1",
            targetHandle: "2",
          } as v0.Edge,
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.edges[0]).toEqual({
        key: "e1",
        source: { node: "n1", param: "1" },
        target: { node: "n2", param: "2" },
      });
    });

    it("should move edge.data segments/color/variant into the props record", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            data: {
              segments: [{ direction: "x", length: 10 }],
              color: "#00ff00",
              variant: "pipe",
            },
          } as unknown as v0.Edge,
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.props.e1).toMatchObject({
        segments: [{ direction: "x", length: 10 }],
        color: color.construct("#00ff00"),
        variant: "pipe",
      });
    });

    it("should add an empty selected array when migrating to v6", () => {
      const migrated = migrateState(v5.ZERO_STATE);
      expect(migrated.selected).toEqual([]);
    });

    it("should widen v5 legend string colors to color.Color when migrating to v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        legend: {
          ...v5.ZERO_STATE.legend,
          colors: { a: "#ff0000", b: "#00ff00" },
        },
      };
      const migrated = migrateState(populated);
      expect(migrated.legend.colors.a).toEqual(color.construct("#ff0000"));
      expect(migrated.legend.colors.b).toEqual(color.construct("#00ff00"));
    });

    it("should migrate a v5 state whose legend has no colors", () => {
      const { colors: _colors, ...legendWithoutColors } = v5.ZERO_STATE.legend;
      const state = {
        ...v5.ZERO_STATE,
        legend: {
          ...legendWithoutColors,
          position: { x: 123, y: 456, units: { x: "px", y: "px" } },
        },
      } as unknown as v5.State;
      const migrated = migrateState(state);
      expect(migrated.legend.colors).toEqual({});
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
