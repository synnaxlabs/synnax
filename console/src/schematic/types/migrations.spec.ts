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
      expect(migrated.configs.n1).toMatchObject({ variant: "valve", color: "#ff0000" });
      expect(migrated.configs.n2).toMatchObject({ variant: "tank" });
      expect(migrated.configs.n1).not.toHaveProperty("key");
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
      expect(migrated.configs.e1).toMatchObject({
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

    it("should preserve non-pipe edge variants when migrating v5 → v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            data: { variant: "electric" },
          } as unknown as v0.Edge,
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.configs.e1).toMatchObject({ variant: "electric" });
    });

    it("should normalize null source/target handles to empty strings", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            sourceHandle: null,
            targetHandle: null,
          } as v0.Edge,
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.edges[0].source).toEqual({ node: "n1", param: "" });
      expect(migrated.edges[0].target).toEqual({ node: "n2", param: "" });
    });

    it("should fall back to pipe defaults when an edge has no data", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [{ key: "e1", source: "n1", target: "n2" } as v0.Edge],
      };
      const migrated = migrateState(populated);
      expect(migrated.configs.e1).toEqual({
        variant: "pipe",
        segments: [],
        color: color.ZERO,
      });
    });

    it("should strip stale selected and unknown fields from migrated nodes", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        nodes: [
          {
            key: "n1",
            position: { x: 10, y: 20 },
            selected: true,
            staleField: "junk",
          } as unknown as v0.Node,
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.nodes[0]).toEqual({ key: "n1", position: { x: 10, y: 20 } });
      expect(migrated.nodes[0]).not.toHaveProperty("selected");
      expect(migrated.nodes[0]).not.toHaveProperty("staleField");
    });

    it("should migrate a populated v0 state through every version to v6", () => {
      const populated: v0.State = {
        ...v0.ZERO_STATE,
        nodes: [{ key: "n1", position: { x: 10, y: 20 } }],
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            sourceHandle: "right",
            targetHandle: "left",
          } as v0.Edge,
        ],
        props: { n1: { key: "valve", color: "#abcdef" } as v0.NodeProps },
      };
      const migrated = migrateState(populated);
      expect(migrated.version).toBe(v6.VERSION);
      expect(migrated.nodes).toEqual([{ key: "n1", position: { x: 10, y: 20 } }]);
      expect(migrated.edges[0]).toEqual({
        key: "e1",
        source: { node: "n1", param: "right" },
        target: { node: "n2", param: "left" },
      });
      expect(migrated.configs.n1).toMatchObject({
        variant: "valve",
        color: "#abcdef",
      });
      expect(migrated.selected).toEqual([]);
      expect(migrated.authority).toBe(1);
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

    it("should migrate the v5 copy buffer to v6 shape", () => {
      const populated: v5.SliceState = {
        ...v5.ZERO_SLICE_STATE,
        copy: {
          pos: { x: 50, y: 60 },
          nodes: [
            {
              key: "n1",
              position: { x: 0, y: 0 },
              selected: true,
            } as unknown as v0.Node,
            { key: "n2", position: { x: 100, y: 0 } } as v0.Node,
          ],
          edges: [
            {
              key: "e1",
              source: "n1",
              target: "n2",
              sourceHandle: "right",
              targetHandle: "left",
              data: { color: "#abcdef", variant: "electric" },
            } as unknown as v0.Edge,
          ],
          props: {
            n1: { key: "valve", color: "#ff0000" } as v0.NodeProps,
            n2: { key: "tank" } as v0.NodeProps,
          },
        },
      };
      const migrated = migrateSlice(populated);
      expect(migrated.copy.pos).toEqual({ x: 50, y: 60 });
      expect(migrated.copy.nodes).toEqual([
        { key: "n1", position: { x: 0, y: 0 } },
        { key: "n2", position: { x: 100, y: 0 } },
      ]);
      expect(migrated.copy.edges[0]).toEqual({
        key: "e1",
        source: { node: "n1", param: "right" },
        target: { node: "n2", param: "left" },
      });
      expect(migrated.copy.configs.n1).toMatchObject({
        variant: "valve",
        color: "#ff0000",
      });
      expect(migrated.copy.configs.n2).toMatchObject({ variant: "tank" });
      expect(migrated.copy.configs.e1).toMatchObject({
        variant: "electric",
        color: color.construct("#abcdef"),
      });
      expect(migrated).not.toHaveProperty("props");
    });

    it("should migrate every schematic in a populated v5 slice", () => {
      const populated: v5.SliceState = {
        ...v5.ZERO_SLICE_STATE,
        schematics: {
          a: {
            ...v5.ZERO_STATE,
            nodes: [{ key: "n1", position: { x: 0, y: 0 } } as v0.Node],
            props: { n1: { key: "valve" } as v0.NodeProps },
          },
          b: {
            ...v5.ZERO_STATE,
            nodes: [{ key: "n2", position: { x: 5, y: 5 } } as v0.Node],
            edges: [
              {
                key: "e1",
                source: "n2",
                target: "n2",
                data: { variant: "secondary" },
              } as unknown as v0.Edge,
            ],
            props: { n2: { key: "tank" } as v0.NodeProps },
          },
        },
      };
      const migrated = migrateSlice(populated);
      expect(migrated.schematics.a.configs.n1).toMatchObject({ variant: "valve" });
      expect(migrated.schematics.a.selected).toEqual([]);
      expect(migrated.schematics.a).not.toHaveProperty("props");
      expect(migrated.schematics.b.configs.n2).toMatchObject({ variant: "tank" });
      expect(migrated.schematics.b.configs.e1).toMatchObject({ variant: "secondary" });
      expect(migrated.schematics.b.edges[0].source).toEqual({ node: "n2", param: "" });
    });
  });
});
