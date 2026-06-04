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
        expect(migrated.version).toBe(ZERO_STATE.version);
        expect(migrated.toolbar.activeTab).toBe(ZERO_STATE.toolbar.activeTab);
        expect(migrated.selected).toEqual([]);
        expect(migrated.legend).toBeDefined();
        expect(migrated.viewport).toBeDefined();
      });
    });

    it("should park v5 graph state into pendingUpload as typed v6 configs", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            sourceHandle: "1",
            targetHandle: "2",
          },
        ],
        props: { n1: { key: "valve", color: "#ff0000" } },
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload).toBeDefined();
      expect(migrated.pendingUpload?.nodes).toHaveLength(1);
      expect(migrated.pendingUpload?.edges).toEqual([
        {
          key: "e1",
          source: { node: "n1", param: "1" },
          target: { node: "n2", param: "2" },
        },
      ]);
      expect(migrated.pendingUpload?.configs.n1).toMatchObject({
        variant: "valve",
        color: "#ff0000",
      });
      expect(migrated.pendingUpload?.configs.n1).not.toHaveProperty("key");
    });

    it("should rename nodePropsZ.key to .variant when migrating v5 → v6", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        props: {
          n1: { key: "valve", color: "#ff0000" },
          n2: { key: "tank" },
        },
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.configs.n1).toMatchObject({
        variant: "valve",
        color: "#ff0000",
      });
      expect(migrated.pendingUpload?.configs.n2).toMatchObject({ variant: "tank" });
    });

    it("should move edge.data segments/color/variant into the configs record", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            data: {
              segments: [
                { direction: "x", length: 10 },
                { direction: "y", length: -50 },
                { direction: "x", length: 30 },
                { direction: "y", length: 10 },
              ],
              color: "#00ff00",
              variant: "pipe",
            },
          },
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.configs.e1).toMatchObject({
        segments: [
          { direction: "y", length: -50 },
          { direction: "x", length: 30 },
        ],
        color: color.construct("#00ff00"),
        variant: "pipe",
      });
    });

    it("should strip legacy stumps from full-path edge segments", () => {
      // Real OX Pre-Valve -> OX MPV edge from a 0.55 schematic: the stored full path
      // includes both stumps, which would double on render and fold a pigtail.
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            sourceHandle: "2",
            targetHandle: "2",
            data: {
              segments: [
                { direction: "x", length: 10 },
                { direction: "y", length: -281.7166395035551 },
                { direction: "x", length: 140.06190790464655 },
                { direction: "y", length: 10 },
              ],
              color: "#004ad6",
              variant: "jacketed",
            },
          },
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.configs.e1).toMatchObject({
        segments: [
          { direction: "y", length: -281.7166395035551 },
          { direction: "x", length: 140.06190790464655 },
        ],
        variant: "jacketed",
      });
    });

    it("should clear degenerate short edges so they auto-route", () => {
      // A single segment shorter than two stumps (real 0.55 edge, 11.88px) has no
      // strippable middle; subtracting a full stump from each end would flip it into a
      // self-crossing spur, so it must be cleared to an empty (auto-routed) edge.
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [
          {
            key: "e1",
            source: "n1",
            target: "n2",
            data: { segments: [{ direction: "y", length: 11.88 }], variant: "pipe" },
          },
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.configs.e1).toMatchObject({ segments: [] });
    });

    it("should add an empty selected array when migrating to v6", () => {
      const migrated = migrateState(v5.ZERO_STATE);
      expect(migrated.selected).toEqual([]);
    });

    it("should preserve legend visibility, position, and parsed colors", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        legend: {
          visible: true,
          position: { x: 123, y: 456, units: { x: "px", y: "px" } },
          colors: { a: "#ff0000", b: "#00ff00" },
        },
      };
      const migrated = migrateState(populated);
      expect(migrated.legend.visible).toBe(true);
      expect(migrated.legend.position).toEqual({
        x: 123,
        y: 456,
        units: { x: "px", y: "px" },
      });
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
      } as v5.State;
      const migrated = migrateState(state);
      expect(migrated.legend.colors).toEqual({});
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
          },
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.configs.e1).toMatchObject({
        variant: "electric",
      });
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
          },
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.edges[0].source).toEqual({
        node: "n1",
        param: "",
      });
      expect(migrated.pendingUpload?.edges[0].target).toEqual({
        node: "n2",
        param: "",
      });
    });

    it("should fall back to pipe defaults when an edge has no data", () => {
      const populated: v5.State = {
        ...v5.ZERO_STATE,
        edges: [{ key: "e1", source: "n1", target: "n2" }],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.configs.e1).toEqual({
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
          },
        ],
      };
      const migrated = migrateState(populated);
      expect(migrated.pendingUpload?.nodes[0]).toEqual({
        key: "n1",
        position: { x: 10, y: 20 },
      });
      expect(migrated.pendingUpload?.nodes[0]).not.toHaveProperty("selected");
      expect(migrated.pendingUpload?.nodes[0]).not.toHaveProperty("staleField");
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
          },
        ],
        props: { n1: { key: "valve", color: "#abcdef" } },
      };
      const migrated = migrateState(populated);
      expect(migrated.version).toBe(v6.VERSION);
      expect(migrated.pendingUpload?.nodes).toEqual([
        { key: "n1", position: { x: 10, y: 20 } },
      ]);
      expect(migrated.pendingUpload?.edges[0]).toEqual({
        key: "e1",
        source: { node: "n1", param: "right" },
        target: { node: "n2", param: "left" },
      });
      expect(migrated.pendingUpload?.configs.n1).toMatchObject({
        variant: "valve",
        color: "#abcdef",
      });
      expect(migrated.selected).toEqual([]);
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

    it("should migrate every schematic in a populated v5 slice into pendingUpload", () => {
      const populated: v5.SliceState = {
        ...v5.ZERO_SLICE_STATE,
        schematics: {
          a: {
            ...v5.ZERO_STATE,
            nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
            props: { n1: { key: "valve" } },
          },
          b: {
            ...v5.ZERO_STATE,
            nodes: [{ key: "n2", position: { x: 5, y: 5 } }],
            edges: [
              {
                key: "e1",
                source: "n2",
                target: "n2",
                data: { variant: "secondary" },
              },
            ],
            props: { n2: { key: "tank" } },
          },
        },
      };
      const migrated = migrateSlice(populated);
      expect(migrated.schematics.a.pendingUpload?.configs.n1).toMatchObject({
        variant: "valve",
      });
      expect(migrated.schematics.a.selected).toEqual([]);
      expect(migrated.schematics.a).not.toHaveProperty("props");
      expect(migrated.schematics.b.pendingUpload?.configs.n2).toMatchObject({
        variant: "tank",
      });
      expect(migrated.schematics.b.pendingUpload?.configs.e1).toMatchObject({
        variant: "secondary",
      });
      expect(migrated.schematics.b.pendingUpload?.edges[0].source).toEqual({
        node: "n2",
        param: "",
      });
    });
  });
});
