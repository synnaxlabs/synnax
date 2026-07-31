// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, type record } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import { type Panel } from "@/platform/panel";
import { createGrantedFluxStore, createTestFluxStore, uniqueName } from "@/testutil";

const V0_ZERO = {
  version: "0.0.0",
  editable: true,
  fitViewOnResize: false,
  snapshot: false,
  remoteCreated: false,
  viewport: { position: { x: 0, y: 0 }, zoom: 1 },
  nodes: [],
  edges: [],
  props: {},
  control: "released",
};
const V1_ZERO = {
  ...V0_ZERO,
  version: "1.0.0",
  legend: { visible: true, position: { x: 50, y: 50, units: { x: "px", y: "px" } } },
};
const V2_ZERO = {
  ...V1_ZERO,
  version: "2.0.0",
  key: "",
  type: "schematic",
  viewportMode: "select",
};
const V3_ZERO = { ...V2_ZERO, version: "3.0.0" };
const V4_ZERO = { ...V3_ZERO, version: "4.0.0", authority: 1 };
const { type: _type, ...V4_ZERO_NO_TYPE } = V4_ZERO;
const V5_ZERO = {
  ...V4_ZERO_NO_TYPE,
  version: "5.0.0",
  mode: "select",
  toolbar: { activeTab: "symbols", selectedSymbolGroup: "general" },
};
const V6_ZERO = {
  version: "6.0.0",
  selected: [],
  controlStatus: "released",
  authority: 1,
  legend: {
    visible: true,
    position: { x: 50, y: 50, units: { x: "px", y: "px" } },
    colors: {},
  },
  toolbar: { activeTab: "symbols", selectedSymbolGroup: "general" },
  editable: false,
  fitViewOnResize: false,
  viewport: { position: { x: 0, y: 0 }, zoom: 1, mode: "select" },
};

describe("schematic state migrations", () => {
  it.each([
    ["0.0.0", V0_ZERO],
    ["1.0.0", V1_ZERO],
    ["2.0.0", V2_ZERO],
    ["3.0.0", V3_ZERO],
    ["4.0.0", V4_ZERO],
    ["5.0.0", V5_ZERO],
    ["6.0.0", V6_ZERO],
  ])("should migrate state from %s to latest", (_version, state) => {
    const migrated = Schematic.anyStateZ.parse(state);
    expect(migrated.version).toBe("6.0.0");
    expect(migrated.toolbar.activeTab).toBe("symbols");
    expect(migrated.selected).toEqual([]);
    expect(migrated.legend).toBeDefined();
    expect(migrated.viewport).toBeDefined();
  });

  it("should park v5 graph state into pendingUpload as typed v6 configs", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
      edges: [
        { key: "e1", source: "n1", target: "n2", sourceHandle: "1", targetHandle: "2" },
      ],
      props: { n1: { key: "valve", color: "#ff0000" } },
    });
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
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      props: { n1: { key: "valve", color: "#ff0000" }, n2: { key: "tank" } },
    });
    expect(migrated.pendingUpload?.configs.n1).toMatchObject({
      variant: "valve",
      color: "#ff0000",
    });
    expect(migrated.pendingUpload?.configs.n2).toMatchObject({ variant: "tank" });
  });

  it("should move edge.data segments/color/variant into the configs record", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
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
    });
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
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
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
    });
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
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      edges: [
        {
          key: "e1",
          source: "n1",
          target: "n2",
          data: { segments: [{ direction: "y", length: 11.88 }], variant: "pipe" },
        },
      ],
    });
    expect(migrated.pendingUpload?.configs.e1).toMatchObject({ segments: [] });
  });

  it("should add an empty selected array when migrating to v6", () => {
    const migrated = Schematic.anyStateZ.parse(V5_ZERO);
    expect(migrated.selected).toEqual([]);
  });

  it("should preserve legend visibility, position, and parsed colors", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      legend: {
        visible: true,
        position: { x: 123, y: 456, units: { x: "px", y: "px" } },
        colors: { a: "#ff0000", b: "#00ff00" },
      },
    });
    expect(migrated.legend.visible).toBe(true);
    expect(migrated.legend.position).toEqual({
      x: 123,
      y: 456,
      units: { x: "px", y: "px" },
      root: { x: "left", y: "top" },
    });
    expect(migrated.legend.colors.a).toEqual(color.construct("#ff0000"));
    expect(migrated.legend.colors.b).toEqual(color.construct("#00ff00"));
  });

  it("should migrate a v5 state whose legend has no colors", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      legend: {
        visible: true,
        position: { x: 123, y: 456, units: { x: "px", y: "px" } },
      },
    });
    expect(migrated.legend.colors).toEqual({});
  });

  it("should preserve non-pipe edge variants when migrating v5 → v6", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      edges: [{ key: "e1", source: "n1", target: "n2", data: { variant: "electric" } }],
    });
    expect(migrated.pendingUpload?.configs.e1).toMatchObject({ variant: "electric" });
  });

  it("should normalize null source/target handles to empty strings", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      edges: [
        {
          key: "e1",
          source: "n1",
          target: "n2",
          sourceHandle: null,
          targetHandle: null,
        },
      ],
    });
    expect(migrated.pendingUpload?.edges[0].source).toEqual({ node: "n1", param: "" });
    expect(migrated.pendingUpload?.edges[0].target).toEqual({ node: "n2", param: "" });
  });

  it("should fall back to pipe defaults when an edge has no data", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      edges: [{ key: "e1", source: "n1", target: "n2" }],
    });
    expect(migrated.pendingUpload?.configs.e1).toEqual({
      variant: "pipe",
      segments: [],
      color: color.ZERO,
    });
  });

  it("should strip stale selected and unknown fields from migrated nodes", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V5_ZERO,
      nodes: [
        { key: "n1", position: { x: 10, y: 20 }, selected: true, staleField: "junk" },
      ],
    });
    expect(migrated.pendingUpload?.nodes[0]).toEqual({
      key: "n1",
      position: { x: 10, y: 20 },
      zIndex: 0,
    });
    expect(migrated.pendingUpload?.nodes[0]).not.toHaveProperty("selected");
    expect(migrated.pendingUpload?.nodes[0]).not.toHaveProperty("staleField");
  });

  it("should migrate a populated v0 state through every version to v6", () => {
    const migrated = Schematic.anyStateZ.parse({
      ...V0_ZERO,
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
    });
    expect(migrated.version).toBe("6.0.0");
    expect(migrated.pendingUpload?.nodes).toEqual([
      { key: "n1", position: { x: 10, y: 20 }, zIndex: 0 },
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

const configsOf = (s: { configs?: unknown }): record.Unknown =>
  typeof s.configs === "object" && s.configs != null ? { ...s.configs } : {};

const LEGACY_V2 = {
  version: "2.0.0",
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  type: "schematic",
  name: "Schematic",
  editable: true,
  fitViewOnResize: false,
  snapshot: false,
  remoteCreated: false,
  control: "released",
  viewport: { position: { x: 0, y: 0 }, zoom: 1 },
  viewportMode: "select",
  legend: { visible: false, position: { x: 50, y: 50, units: { x: "px", y: "px" } } },
  nodes: [
    {
      key: "n1",
      position: { x: -300, y: -3.5 },
      type: "custom",
      width: 230,
      height: 112,
      zIndex: 4,
    },
  ],
  edges: [],
  props: { n1: { key: "valve", color: [28, 28, 28, 1] } },
};

const TYPED_EXPORT = {
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  name: "Schematic",
  type: "schematic",
  version: "6.0.0",
  snapshot: false,
  nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
  edges: [],
  configs: { n1: { variant: "valve", color: [28, 28, 28, 1] } },
};

describe("schematic import", () => {
  describe("parseImport", () => {
    it("should migrate a legacy console export, preserving every symbol config", () => {
      const out = Schematic.parseImport(LEGACY_V2, undefined);
      expect(out.nodes).toHaveLength(1);
      expect(configsOf(out).n1).toMatchObject({ variant: "valve" });
    });

    it("should import a typed schematic export directly, preserving configs", () => {
      const out = Schematic.parseImport(TYPED_EXPORT, undefined);
      expect(out.nodes).toHaveLength(1);
      expect(configsOf(out).n1).toMatchObject({ variant: "valve" });
    });

    it("should reject a legacy state that carries no graph data", () => {
      expect(() => Schematic.parseImport(V6_ZERO, undefined)).toThrow(
        "Imported schematic has no graph data",
      );
    });
  });

  describe("ingest", () => {
    it("creates the schematic on the cluster and opens its tab", async () => {
      const client = createTestClient();
      const store = await createGrantedFluxStore(
        client,
        schematic.TYPE_ONTOLOGY_ID,
        "update",
      );
      const project = await client.projects.create({
        name: uniqueName("proj"),
        layout: {},
      });
      const openTab = vi.fn<Panel.OpenTab>();
      const name = uniqueName("imported");
      const id = await Schematic.ingest(TYPED_EXPORT, {
        name,
        openTab,
        store,
        client,
        projectKey: project.key,
        fileName: "test.json",
      });
      if (id == null) throw new Error("ingest did not return an ontology id");
      expect(openTab).toHaveBeenCalledTimes(1);
      expect(openTab).toHaveBeenCalledWith({ variant: "resource", resource: id });
      const created = await client.schematics.retrieve({ key: id.key });
      expect(created.name).toBe(name);
      expect(created.nodes).toHaveLength(1);
      expect(store.schematics.get(id.key)?.name).toBe(name);
    });

    it("rejects the import when the permission cache has no grant", async () => {
      const store = createTestFluxStore(null);
      await expect(
        Schematic.ingest(TYPED_EXPORT, {
          name: "denied",
          openTab: vi.fn<Panel.OpenTab>(),
          store,
          client: null,
          projectKey: "project-1",
          fileName: "test.json",
        }),
      ).rejects.toThrow("You do not have permission to import schematics");
    });
  });
});
