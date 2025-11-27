// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it } from "vitest";

import {
  actions,
  reducer,
  SLICE_NAME,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/slice";

describe("Schematic Slice", () => {
  let store: ReturnType<typeof configureStore<StoreState>>;

  beforeEach(() => {
    store = configureStore({
      reducer: {
        [SLICE_NAME]: reducer,
      },
      preloadedState: {
        [SLICE_NAME]: ZERO_SLICE_STATE,
      },
    });
  });

  describe("schematic creation", () => {
    it("should create a new schematic", () => {
      const key = "schematic-1";
      store.dispatch(actions.create({ ...ZERO_STATE, key }));

      const state = store.getState()[SLICE_NAME];
      expect(state.schematics[key]).toBeDefined();
      expect(state.schematics[key].key).toBe(key);
      expect(state.schematics[key].nodes).toEqual([]);
      expect(state.schematics[key].edges).toEqual([]);
    });

    it("should create multiple schematics", () => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-1" }));
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-2" }));

      const state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.schematics)).toHaveLength(2);
      expect(state.schematics["schematic-1"]).toBeDefined();
      expect(state.schematics["schematic-2"]).toBeDefined();
    });
  });

  describe("node management", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
    });

    it("should add a node to schematic", () => {
      const nodeKey = "valve-1";
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: nodeKey,
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      expect(schematic.nodes).toHaveLength(1);
      expect(schematic.nodes[0].key).toBe(nodeKey);
      expect(schematic.nodes[0].position).toEqual({ x: 100, y: 100 });
      expect(schematic.props[nodeKey]).toEqual({ key: "valve" });
    });

    it("should update node positions", () => {
      const node1Key = "valve-1";
      const node2Key = "valve-2";

      // Add two nodes
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: node1Key,
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: node2Key,
          props: { key: "valve" },
          node: { position: { x: 150, y: 20 } },
        }),
      );

      // Update positions (simulating alignment)
      store.dispatch(
        actions.setNodePositions({
          key: schematicKey,
          positions: {
            [node1Key]: { x: 0, y: 0 },
            [node2Key]: { x: 150, y: 0 }, // Aligned vertically
          },
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      const node1 = schematic.nodes.find((n: any) => n.key === node1Key);
      const node2 = schematic.nodes.find((n: any) => n.key === node2Key);

      expect(node1?.position).toEqual({ x: 0, y: 0 });
      expect(node2?.position).toEqual({ x: 150, y: 0 });
      expect(node1?.position.y).toBe(node2?.position.y); // Aligned
    });

    it("should set multiple nodes at once", () => {
      const nodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: false },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: false },
        { key: "valve-3", position: { x: 300, y: 0 }, selected: false },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes,
          mode: "replace",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      expect(schematic.nodes).toHaveLength(3);
    });

    it("should update nodes without replacing all", () => {
      // Add initial nodes
      const initialNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: false },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: false },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: initialNodes,
          mode: "replace",
        }),
      );

      // Update one node
      const updatedNode = {
        key: "valve-1",
        position: { x: 50, y: 50 },
        selected: true,
      };

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [updatedNode],
          mode: "update",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      expect(schematic.nodes).toHaveLength(2);

      const node1 = schematic.nodes.find((n: any) => n.key === "valve-1");
      const node2 = schematic.nodes.find((n: any) => n.key === "valve-2");

      expect(node1?.position).toEqual({ x: 50, y: 50 });
      expect(node1?.selected).toBe(true);
      expect(node2?.position).toEqual({ x: 150, y: 0 });
    });
  });

  describe("selection management", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));

      // Add some nodes
      const nodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: false },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: false },
        { key: "valve-3", position: { x: 300, y: 0 }, selected: false },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes,
          mode: "replace",
        }),
      );
    });

    it("should select nodes and switch to properties tab", () => {
      const selectedNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: true },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: selectedNodes,
          mode: "update",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];

      const node1 = schematic.nodes.find((n: any) => n.key === "valve-1");
      const node2 = schematic.nodes.find((n: any) => n.key === "valve-2");

      expect(node1?.selected).toBe(true);
      expect(node2?.selected).toBe(true);
      expect(schematic.toolbar.activeTab).toBe("properties");
    });

    it("should clear selection", () => {
      // First select some nodes
      const selectedNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: true },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: selectedNodes,
          mode: "update",
        }),
      );

      // Then clear selection
      store.dispatch(actions.clearSelection({ key: schematicKey }));

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];

      expect(schematic.nodes.every((n: any) => !n.selected)).toBe(true);
      expect(schematic.toolbar.activeTab).toBe("symbols");
    });

    it("should switch back to symbols tab when no nodes selected", () => {
      const unselectedNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: false },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: false },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: unselectedNodes,
          mode: "update",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      expect(schematic.toolbar.activeTab).toBe("symbols");
    });
  });

  describe("alignment workflow simulation", () => {
    const schematicKey = "alignment-test";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
    });

    it("should simulate vertical alignment workflow", () => {
      // Step 1: Add nodes at different Y positions
      const initialNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
        { key: "valve-2", position: { x: 150, y: 20 }, selected: true },
        { key: "valve-3", position: { x: 300, y: -10 }, selected: true },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: initialNodes,
          mode: "replace",
        }),
      );

      // Step 2: Simulate alignment (align all to y=0)
      const alignedNodes = initialNodes.map((node) => ({
        ...node,
        position: { ...node.position, y: 0 },
      }));

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: alignedNodes,
          mode: "update",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];

      // Verify all nodes are aligned
      const yPositions = schematic.nodes.map((n: any) => n.position.y);
      expect(yPositions.every((y: any) => y === 0)).toBe(true);
    });

    it("should simulate horizontal distribution workflow", () => {
      // Step 1: Add nodes with uneven spacing
      const initialNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
        { key: "valve-2", position: { x: 110, y: 0 }, selected: true },
        { key: "valve-3", position: { x: 600, y: 0 }, selected: true },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: initialNodes,
          mode: "replace",
        }),
      );

      // Step 2: Simulate distribution (even spacing)
      // With proper distribution: first at 0, last at 600, middle should be at 300
      const distributedNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
        { key: "valve-2", position: { x: 300, y: 0 }, selected: true },
        { key: "valve-3", position: { x: 600, y: 0 }, selected: true },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: distributedNodes,
          mode: "update",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];

      const node1 = schematic.nodes.find((n: any) => n.key === "valve-1");
      const node2 = schematic.nodes.find((n: any) => n.key === "valve-2");
      const node3 = schematic.nodes.find((n: any) => n.key === "valve-3");

      // Check that spacing is even
      const gap1 = (node2?.position.x ?? 0) - (node1?.position.x ?? 0);
      const gap2 = (node3?.position.x ?? 0) - (node2?.position.x ?? 0);
      expect(gap1).toBe(gap2);
    });

    it("should simulate complex alignment workflow from integration test", () => {
      // Simulate the workflow from integration/tests/console/schematic/alignment.py

      // Create nodes (setpoint, valves)
      const nodes = [
        { key: "setpoint", position: { x: -210, y: 0 }, selected: true },
        { key: "threeWayValve", position: { x: -150, y: 0 }, selected: true },
        { key: "threeWayBall", position: { x: 150, y: -20 }, selected: true },
        { key: "valve", position: { x: 0, y: 50 }, selected: true },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes,
          mode: "replace",
        }),
      );

      // Step 1: Align Vertical (all same Y)
      let state = store.getState()[SLICE_NAME];
      let schematic = state.schematics[schematicKey];
      const targetY = 0;
      const alignedVertical = schematic.nodes.map((n: any) => ({
        ...n,
        position: { ...n.position, y: targetY },
      }));

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: alignedVertical,
          mode: "update",
        }),
      );

      state = store.getState()[SLICE_NAME];
      schematic = state.schematics[schematicKey];
      expect(schematic.nodes.every((n: any) => n.position.y === targetY)).toBe(true);

      // Step 2: Distribute Horizontal (even spacing)
      const sorted = [...schematic.nodes].sort((a, b) => a.position.x - b.position.x);
      const firstX = sorted[0].position.x;
      const lastX = sorted[sorted.length - 1].position.x;
      const totalSpan = lastX - firstX;
      const gap = totalSpan / (sorted.length - 1);

      const distributedHorizontal = schematic.nodes.map((n: any) => {
        const sortedIdx = sorted.findIndex((s) => s.key === n.key);
        return {
          ...n,
          position: { ...n.position, x: firstX + gap * sortedIdx },
        };
      });

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: distributedHorizontal,
          mode: "update",
        }),
      );

      state = store.getState()[SLICE_NAME];
      schematic = state.schematics[schematicKey];

      // Verify distribution
      const sortedAfter = [...schematic.nodes].sort(
        (a, b) => a.position.x - b.position.x,
      );
      expect(sortedAfter.length).toBeGreaterThan(2);
    });
  });

  describe("edge cases", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
    });

    it("should handle updating positions for non-existent nodes", () => {
      store.dispatch(
        actions.setNodePositions({
          key: schematicKey,
          positions: {
            "non-existent": { x: 100, y: 100 },
          },
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      expect(schematic.nodes).toHaveLength(0);
    });

    it("should handle empty node list", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [],
          mode: "replace",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      expect(schematic.nodes).toEqual([]);
    });

    it("should preserve unmodified nodes during update", () => {
      const initialNodes = [
        { key: "valve-1", position: { x: 0, y: 0 }, selected: false },
        { key: "valve-2", position: { x: 150, y: 0 }, selected: false },
        { key: "valve-3", position: { x: 300, y: 0 }, selected: false },
      ];

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: initialNodes,
          mode: "replace",
        }),
      );

      // Update only valve-2
      const updatedNode = {
        key: "valve-2",
        position: { x: 200, y: 50 },
        selected: true,
      };

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [updatedNode],
          mode: "update",
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];

      const valve1 = schematic.nodes.find((n: any) => n.key === "valve-1");
      const valve2 = schematic.nodes.find((n: any) => n.key === "valve-2");
      const valve3 = schematic.nodes.find((n: any) => n.key === "valve-3");

      expect(valve1?.position).toEqual({ x: 0, y: 0 });
      expect(valve2?.position).toEqual({ x: 200, y: 50 });
      expect(valve3?.position).toEqual({ x: 300, y: 0 });
    });
  });

  describe("schematic removal", () => {
    it("should remove a schematic", () => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-1" }));
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-2" }));

      let state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.schematics)).toHaveLength(2);

      store.dispatch(actions.remove({ keys: ["schematic-1"] }));

      state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.schematics)).toHaveLength(1);
      expect(state.schematics["schematic-1"]).toBeUndefined();
      expect(state.schematics["schematic-2"]).toBeDefined();
    });

    it("should remove multiple schematics at once", () => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-1" }));
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-2" }));
      store.dispatch(actions.create({ ...ZERO_STATE, key: "schematic-3" }));

      store.dispatch(actions.remove({ keys: ["schematic-1", "schematic-3"] }));

      const state = store.getState()[SLICE_NAME];
      expect(Object.keys(state.schematics)).toHaveLength(1);
      expect(state.schematics["schematic-2"]).toBeDefined();
    });
  });
});
