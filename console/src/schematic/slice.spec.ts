// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { type Diagram } from "@synnaxlabs/pluto";
import { beforeEach, describe, expect, it } from "vitest";

import {
  selectCanGroup,
  selectCanUngroup,
  selectNodeProps,
  selectSelectedElementDigests,
} from "@/schematic/selectors";
import {
  actions,
  reducer,
  SLICE_NAME,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/schematic/slice";
import { type NodeProps } from "@/schematic/types";

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
          positions: [
            [node1Key, { x: 0, y: 0 }],
            [node2Key, { x: 150, y: 0 }], // Aligned vertically
          ],
        }),
      );

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      const node1 = schematic.nodes.find((n: Diagram.Node) => n.key === node1Key);
      const node2 = schematic.nodes.find((n: Diagram.Node) => n.key === node2Key);

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

      const node1 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-1");
      const node2 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-2");

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

      const node1 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-1");
      const node2 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-2");

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

      expect(schematic.nodes.every((n: Diagram.Node) => !n.selected)).toBe(true);
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
      const yPositions = schematic.nodes.map((n: Diagram.Node) => n.position.y);
      expect(yPositions.every((y) => y === 0)).toBe(true);
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

      const node1 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-1");
      const node2 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-2");
      const node3 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-3");

      // Check that spacing is even
      const gap1 = (node2?.position.x ?? 0) - (node1?.position.x ?? 0);
      const gap2 = (node3?.position.x ?? 0) - (node2?.position.x ?? 0);
      expect(gap1).toBe(gap2);
    });

    it("should handle vertical alignment followed by horizontal distribution", () => {
      // Create nodes at different positions
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
      const alignedVertical = schematic.nodes.map((n: Diagram.Node) => ({
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
      expect(schematic.nodes.every((n: Diagram.Node) => n.position.y === targetY)).toBe(
        true,
      );

      // Step 2: Distribute Horizontal (even spacing)
      const sorted = [...schematic.nodes].sort((a, b) => a.position.x - b.position.x);
      const firstX = sorted[0].position.x;
      const lastX = sorted[sorted.length - 1].position.x;
      const totalSpan = lastX - firstX;
      const gap = totalSpan / (sorted.length - 1);

      const distributedHorizontal = schematic.nodes.map((n: Diagram.Node) => {
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
          positions: [["non-existent", { x: 100, y: 100 }]],
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

      const valve1 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-1");
      const valve2 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-2");
      const valve3 = schematic.nodes.find((n: Diagram.Node) => n.key === "valve-3");

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

  describe("copy, cut, and paste", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve", color: "#ff0000" },
          node: { position: { x: 100, y: 200 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve", color: "#00ff00" },
          node: { position: { x: 300, y: 400 } },
        }),
      );
      // Select both nodes
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 100, y: 200 }, selected: true },
            { key: "valve-2", position: { x: 300, y: 400 }, selected: true },
          ],
          mode: "update",
        }),
      );
    });

    describe("copySelection", () => {
      it("should copy selected nodes into the copy buffer", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.copy.nodes).toHaveLength(2);
        expect(state.copy.nodes.map((n) => n.key)).toContain("valve-1");
        expect(state.copy.nodes.map((n) => n.key)).toContain("valve-2");
      });

      it("should copy node props into the copy buffer", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.copy.props["valve-1"]).toEqual({ key: "valve", color: "#ff0000" });
        expect(state.copy.props["valve-2"]).toEqual({ key: "valve", color: "#00ff00" });
      });

      it("should compute the centroid position of copied nodes", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.copy.pos).toEqual({ x: 200, y: 300 });
      });

      it("should not remove nodes from the schematic", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.schematics[schematicKey].nodes).toHaveLength(2);
      });

      it("should not copy unselected nodes", () => {
        // Deselect valve-2
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [{ key: "valve-2", position: { x: 300, y: 400 }, selected: false }],
            mode: "update",
          }),
        );

        store.dispatch(actions.copySelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.copy.nodes).toHaveLength(1);
        expect(state.copy.nodes[0].key).toBe("valve-1");
      });

      it("should handle no selected nodes", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.copy.nodes).toHaveLength(0);
        expect(state.copy.pos).toEqual({ x: 0, y: 0 });
      });
    });

    describe("cutSelection", () => {
      it("should copy selected nodes into the copy buffer", () => {
        store.dispatch(actions.cutSelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.copy.nodes).toHaveLength(2);
        expect(state.copy.props["valve-1"]).toEqual({ key: "valve", color: "#ff0000" });
      });

      it("should remove cut nodes from the schematic", () => {
        store.dispatch(actions.cutSelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.schematics[schematicKey].nodes).toHaveLength(0);
      });

      it("should remove cut node props from the schematic", () => {
        store.dispatch(actions.cutSelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.schematics[schematicKey].props["valve-1"]).toBeUndefined();
        expect(state.schematics[schematicKey].props["valve-2"]).toBeUndefined();
      });

      it("should not remove unselected nodes", () => {
        // Deselect valve-2
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [{ key: "valve-2", position: { x: 300, y: 400 }, selected: false }],
            mode: "update",
          }),
        );

        store.dispatch(actions.cutSelection({ key: schematicKey }));

        const state = store.getState()[SLICE_NAME];
        expect(state.schematics[schematicKey].nodes).toHaveLength(1);
        expect(state.schematics[schematicKey].nodes[0].key).toBe("valve-2");
      });
    });

    describe("pasteSelection", () => {
      it("should paste copied nodes at the given position", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({
            key: schematicKey,
            pos: { x: 400, y: 500 },
          }),
        );

        const state = store.getState()[SLICE_NAME];
        // Original 2 + pasted 2
        expect(state.schematics[schematicKey].nodes).toHaveLength(4);
      });

      it("should generate new keys for pasted nodes", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({
            key: schematicKey,
            pos: { x: 200, y: 300 },
          }),
        );

        const state = store.getState()[SLICE_NAME];
        const keys = state.schematics[schematicKey].nodes.map((n) => n.key);
        const uniqueKeys = new Set(keys);
        expect(uniqueKeys.size).toBe(keys.length);
      });

      it("should offset pasted nodes relative to the paste position", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));
        // Centroid is (200, 300), paste at (400, 500) → offset (200, 200)
        store.dispatch(
          actions.pasteSelection({
            key: schematicKey,
            pos: { x: 400, y: 500 },
          }),
        );

        const state = store.getState()[SLICE_NAME];
        const pastedNodes = state.schematics[schematicKey].nodes.filter(
          (n) => n.key !== "valve-1" && n.key !== "valve-2",
        );
        const positions = pastedNodes.map((n) => n.position);
        // valve-1 was at (100,200), offset by (200,200) → (300,400)
        // valve-2 was at (300,400), offset by (200,200) → (500,600)
        expect(positions).toContainEqual({ x: 300, y: 400 });
        expect(positions).toContainEqual({ x: 500, y: 600 });
      });

      it("should select only the pasted nodes", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({
            key: schematicKey,
            pos: { x: 200, y: 300 },
          }),
        );

        const state = store.getState()[SLICE_NAME];
        const nodes = state.schematics[schematicKey].nodes;
        const originalNodes = nodes.filter(
          (n) => n.key === "valve-1" || n.key === "valve-2",
        );
        const pastedNodes = nodes.filter(
          (n) => n.key !== "valve-1" && n.key !== "valve-2",
        );
        expect(originalNodes.every((n) => !n.selected)).toBe(true);
        expect(pastedNodes.every((n) => n.selected)).toBe(true);
      });

      it("should copy props to pasted nodes", () => {
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({
            key: schematicKey,
            pos: { x: 200, y: 300 },
          }),
        );

        const state = store.getState()[SLICE_NAME];
        const pastedNodes = state.schematics[schematicKey].nodes.filter(
          (n) => n.key !== "valve-1" && n.key !== "valve-2",
        );
        pastedNodes.forEach((node) => {
          expect(state.schematics[schematicKey].props[node.key]).toBeDefined();
          expect(state.schematics[schematicKey].props[node.key].key).toBe("valve");
        });
      });
    });

    describe("cut then paste", () => {
      it("should move nodes from one position to another", () => {
        store.dispatch(actions.cutSelection({ key: schematicKey }));

        let state = store.getState()[SLICE_NAME];
        expect(state.schematics[schematicKey].nodes).toHaveLength(0);

        store.dispatch(
          actions.pasteSelection({
            key: schematicKey,
            pos: { x: 500, y: 600 },
          }),
        );

        state = store.getState()[SLICE_NAME];
        expect(state.schematics[schematicKey].nodes).toHaveLength(2);
        expect(state.schematics[schematicKey].nodes.every((n) => n.selected)).toBe(
          true,
        );
      });
    });
  });

  describe("off-page reference page prop", () => {
    const schematicKey = "test-schematic";
    const nodeKey = "opr-1";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: nodeKey,
          props: { key: "offPageReference", page: "" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
    });

    it("should store the page prop on an off-page reference element", () => {
      const props = selectNodeProps(store.getState(), schematicKey, nodeKey);
      expect(props).toBeDefined();
      expect(props?.key).toBe("offPageReference");
      expect(props?.page).toBe("");
    });

    it("should update the page prop via setElementProps", () => {
      const targetPage = "target-schematic-key";
      store.dispatch(
        actions.setElementProps({
          layoutKey: schematicKey,
          key: nodeKey,
          props: { key: "offPageReference", page: targetPage },
        }),
      );

      const props = selectNodeProps(store.getState(), schematicKey, nodeKey);
      expect(props?.page).toBe(targetPage);
    });

    it("should clear the page prop by setting it to empty string", () => {
      store.dispatch(
        actions.setElementProps({
          layoutKey: schematicKey,
          key: nodeKey,
          props: { key: "offPageReference", page: "some-page" },
        }),
      );
      store.dispatch(
        actions.setElementProps({
          layoutKey: schematicKey,
          key: nodeKey,
          props: { key: "offPageReference", page: "" },
        }),
      );

      const props = selectNodeProps(store.getState(), schematicKey, nodeKey);
      expect(props?.page).toBe("");
    });

    it("should preserve the page prop when other props change", () => {
      store.dispatch(
        actions.setElementProps({
          layoutKey: schematicKey,
          key: nodeKey,
          props: { key: "offPageReference", page: "target-page", color: "#ff0000" },
        }),
      );
      store.dispatch(
        actions.setElementProps({
          layoutKey: schematicKey,
          key: nodeKey,
          props: { key: "offPageReference", page: "target-page", color: "#00ff00" },
        }),
      );

      const props = selectNodeProps(store.getState(), schematicKey, nodeKey);
      expect(props?.page).toBe("target-page");
      expect(props?.color).toBe("#00ff00");
    });
  });

  describe("groupSelection", () => {
    const schematicKey = "test-schematic";
    const groupProps: NodeProps = { key: "group" } as NodeProps;

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve", color: "#ff0000" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve", color: "#00ff00" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-3",
          props: { key: "valve", color: "#0000ff" },
          node: { position: { x: 200, y: 200 } },
        }),
      );
    });

    it("should create a group node containing all selected non-group nodes", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      const state = store.getState()[SLICE_NAME];
      const schematic = state.schematics[schematicKey];
      const groupNode = schematic.nodes.find(
        (n) => (schematic.props[n.key] as NodeProps)?.key === "group",
      );
      expect(groupNode).toBeDefined();
      const allProps = schematic.props as Record<string, NodeProps>;
      expect(allProps["valve-1"].groupId).toBe(groupNode!.key);
      expect(allProps["valve-2"].groupId).toBe(groupNode!.key);
      expect(allProps["valve-3"].groupId).toBeUndefined();
    });

    it("should not group when fewer than 2 non-group nodes are selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );
      const before = store.getState()[SLICE_NAME];
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes).toEqual(
        before.schematics[schematicKey].nodes,
      );
    });

    it("should not group when no nodes are selected", () => {
      const before = store.getState()[SLICE_NAME];
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes.length).toBe(
        before.schematics[schematicKey].nodes.length,
      );
    });

    it("should set the group node zIndex to 0 and mark it selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      const state = store.getState()[SLICE_NAME];
      const groupNode = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      );
      expect(groupNode?.zIndex).toBe(0);
      expect(groupNode?.selected).toBe(true);
    });

    it("should store the provided props with calculated dimensions", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      const state = store.getState()[SLICE_NAME];
      const groupNode = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      );
      const props = state.schematics[schematicKey].props[groupNode!.key];
      expect(props.key).toBe("group");
      const dims = (props as Record<string, unknown>).dimensions as {
        width: number;
        height: number;
      };
      expect(dims).toBeDefined();
      expect(dims.width).toBeGreaterThan(0);
      expect(dims.height).toBeGreaterThan(0);
    });

    it("should dissolve a previously-selected group when re-grouping", () => {
      // Create initial group
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      let state = store.getState()[SLICE_NAME];
      const oldGroupNode = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      );
      const oldGroupKey = oldGroupNode!.key;

      // Select the old group + valve-3 and re-group
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: oldGroupKey, position: oldGroupNode!.position, selected: true },
            { key: "valve-3", position: { x: 200, y: 200 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      state = store.getState()[SLICE_NAME];
      const allProps = state.schematics[schematicKey].props as Record<
        string,
        NodeProps
      >;
      // Old group should be removed
      expect(allProps[oldGroupKey]).toBeUndefined();
      expect(
        state.schematics[schematicKey].nodes.find((n) => n.key === oldGroupKey),
      ).toBeUndefined();
      // valve-1, valve-2, valve-3 should all be in the new group
      const newGroupNode = state.schematics[schematicKey].nodes.find(
        (n) => allProps[n.key]?.key === "group",
      );
      expect(newGroupNode).toBeDefined();
      expect(allProps["valve-1"].groupId).toBe(newGroupNode!.key);
      expect(allProps["valve-2"].groupId).toBe(newGroupNode!.key);
      expect(allProps["valve-3"].groupId).toBe(newGroupNode!.key);
    });
  });

  describe("ungroupSelection", () => {
    const schematicKey = "test-schematic";
    const groupProps: NodeProps = { key: "group" } as NodeProps;

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-3",
          props: { key: "valve" },
          node: { position: { x: 200, y: 200 } },
        }),
      );
      // Create a group with valve-1 and valve-2
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));
    });

    it("should remove the group node and clear groupId when the group node is selected", () => {
      const state = store.getState()[SLICE_NAME];
      const allProps = state.schematics[schematicKey].props as Record<
        string,
        NodeProps
      >;
      const groupKey = state.schematics[schematicKey].nodes.find(
        (n) => allProps[n.key]?.key === "group",
      )!.key;

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            {
              key: groupKey,
              position: state.schematics[schematicKey].nodes.find(
                (n) => n.key === groupKey,
              )!.position,
              selected: true,
            },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.ungroupSelection({ key: schematicKey }));

      const after = store.getState()[SLICE_NAME];
      const afterProps = after.schematics[schematicKey].props as Record<
        string,
        NodeProps
      >;
      expect(afterProps[groupKey]).toBeUndefined();
      expect(
        after.schematics[schematicKey].nodes.find((n) => n.key === groupKey),
      ).toBeUndefined();
      expect(afterProps["valve-1"].groupId).toBeUndefined();
      expect(afterProps["valve-2"].groupId).toBeUndefined();
    });

    it("should remove the group when a member node is selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );
      store.dispatch(actions.ungroupSelection({ key: schematicKey }));

      const state = store.getState()[SLICE_NAME];
      const allProps = state.schematics[schematicKey].props as Record<
        string,
        NodeProps
      >;
      expect(allProps["valve-1"].groupId).toBeUndefined();
      expect(allProps["valve-2"].groupId).toBeUndefined();
      const groupNodes = state.schematics[schematicKey].nodes.filter(
        (n) => allProps[n.key]?.key === "group",
      );
      expect(groupNodes).toHaveLength(0);
    });

    it("should do nothing when no selected nodes belong to a group", () => {
      // Deselect everything, select only ungrouped valve-3
      store.dispatch(actions.clearSelection({ key: schematicKey }));
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-3", position: { x: 200, y: 200 }, selected: true }],
          mode: "update",
        }),
      );

      const before = store.getState()[SLICE_NAME];
      store.dispatch(actions.ungroupSelection({ key: schematicKey }));
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes.length).toBe(
        before.schematics[schematicKey].nodes.length,
      );
    });

    it("should preserve unrelated nodes after ungrouping", () => {
      const state = store.getState()[SLICE_NAME];
      const allProps = state.schematics[schematicKey].props as Record<
        string,
        NodeProps
      >;
      const groupKey = state.schematics[schematicKey].nodes.find(
        (n) => allProps[n.key]?.key === "group",
      )!.key;

      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            {
              key: groupKey,
              position: state.schematics[schematicKey].nodes.find(
                (n) => n.key === groupKey,
              )!.position,
              selected: true,
            },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.ungroupSelection({ key: schematicKey }));

      const after = store.getState()[SLICE_NAME];
      expect(
        after.schematics[schematicKey].nodes.find((n) => n.key === "valve-3"),
      ).toBeDefined();
    });
  });

  describe("selectCanGroup", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
    });

    it("should return true when 2 or more non-group nodes are selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      expect(selectCanGroup(store.getState(), schematicKey)).toBe(true);
    });

    it("should return false when only 1 non-group node is selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );
      expect(selectCanGroup(store.getState(), schematicKey)).toBe(false);
    });

    it("should return false when no nodes are selected", () => {
      expect(selectCanGroup(store.getState(), schematicKey)).toBe(false);
    });

    it("should exclude group-type nodes from the count", () => {
      // Create a group, then select only the group node + 1 valve
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(
        actions.groupSelection({
          key: schematicKey,
          props: { key: "group" } as NodeProps,
        }),
      );
      const state = store.getState()[SLICE_NAME];
      const groupKey = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      )!.key;

      // Clear then select only group node
      store.dispatch(actions.clearSelection({ key: schematicKey }));
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            {
              key: groupKey,
              position: state.schematics[schematicKey].nodes.find(
                (n) => n.key === groupKey,
              )!.position,
              selected: true,
            },
          ],
          mode: "update",
        }),
      );
      expect(selectCanGroup(store.getState(), schematicKey)).toBe(false);
    });

    it("should return false for a nonexistent schematic", () => {
      expect(selectCanGroup(store.getState(), "nonexistent")).toBe(false);
    });
  });

  describe("selectCanUngroup", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
    });

    it("should return true when a group-type node is selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(
        actions.groupSelection({
          key: schematicKey,
          props: { key: "group" } as NodeProps,
        }),
      );
      // Group node is auto-selected after groupSelection
      expect(selectCanUngroup(store.getState(), schematicKey)).toBe(true);
    });

    it("should return true when a node with a groupId is selected", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(
        actions.groupSelection({
          key: schematicKey,
          props: { key: "group" } as NodeProps,
        }),
      );
      // Clear selection, then select just a member
      store.dispatch(actions.clearSelection({ key: schematicKey }));
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );
      expect(selectCanUngroup(store.getState(), schematicKey)).toBe(true);
    });

    it("should return false when selected nodes have no group association", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );
      expect(selectCanUngroup(store.getState(), schematicKey)).toBe(false);
    });

    it("should return false when no nodes are selected", () => {
      expect(selectCanUngroup(store.getState(), schematicKey)).toBe(false);
    });

    it("should return false for a nonexistent schematic", () => {
      expect(selectCanUngroup(store.getState(), "nonexistent")).toBe(false);
    });
  });

  describe("selectSelectedElementDigests — group filtering", () => {
    const schematicKey = "test-schematic";

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-3",
          props: { key: "valve" },
          node: { position: { x: 200, y: 200 } },
        }),
      );
      // Create group with valve-1 and valve-2
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(
        actions.groupSelection({
          key: schematicKey,
          props: { key: "group" } as NodeProps,
        }),
      );
    });

    it("should filter out group members when their group box is also selected", () => {
      // After groupSelection, the group node is selected. Members' groupId points
      // to the group, so valve-1 and valve-2 should be filtered out.
      const state = store.getState()[SLICE_NAME];
      const groupKey = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      )!.key;

      // Select group node + valve-1 (a member)
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            {
              key: groupKey,
              position: state.schematics[schematicKey].nodes.find(
                (n) => n.key === groupKey,
              )!.position,
              selected: true,
            },
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
          ],
          mode: "update",
        }),
      );

      const digests = selectSelectedElementDigests(store.getState(), schematicKey);
      const keys = digests.map((d) => d.key);
      expect(keys).toContain(groupKey);
      expect(keys).not.toContain("valve-1");
    });

    it("should include group members when their group box is NOT selected", () => {
      // Deselect everything, select only valve-1 (a member)
      store.dispatch(actions.clearSelection({ key: schematicKey }));
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );

      const digests = selectSelectedElementDigests(store.getState(), schematicKey);
      const keys = digests.map((d) => d.key);
      expect(keys).toContain("valve-1");
    });

    it("should include ungrouped nodes normally", () => {
      store.dispatch(actions.clearSelection({ key: schematicKey }));
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-3", position: { x: 200, y: 200 }, selected: true }],
          mode: "update",
        }),
      );

      const digests = selectSelectedElementDigests(store.getState(), schematicKey);
      expect(digests).toEqual([{ key: "valve-3", type: "node" }]);
    });

    it("should return empty array for non-existent schematic", () => {
      expect(selectSelectedElementDigests(store.getState(), "nonexistent")).toEqual([]);
    });
  });

  describe("copy, cut, and paste — grouped nodes", () => {
    const schematicKey = "test-schematic";
    const groupProps: NodeProps = { key: "group" } as NodeProps;
    let groupKey: string;

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve", color: "#ff0000" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve", color: "#00ff00" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-3",
          props: { key: "valve", color: "#0000ff" },
          node: { position: { x: 200, y: 200 } },
        }),
      );
      // Create group with valve-1 and valve-2
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      const state = store.getState()[SLICE_NAME];
      groupKey = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      )!.key;
    });

    describe("copySelection", () => {
      it("should copy unselected group members when their group node is selected", () => {
        // Clear selection, select only the group node
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const after = store.getState()[SLICE_NAME];
        const copyKeys = after.copy.nodes.map((n) => n.key);
        expect(copyKeys).toContain(groupKey);
        expect(copyKeys).toContain("valve-1");
        expect(copyKeys).toContain("valve-2");
        expect(copyKeys).not.toContain("valve-3");
      });

      it("should copy props for both group node and pulled-in members", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const after = store.getState()[SLICE_NAME];
        expect(after.copy.props[groupKey]).toBeDefined();
        expect(after.copy.props["valve-1"]).toBeDefined();
        expect(after.copy.props["valve-2"]).toBeDefined();
      });

      it("should not pull in members of unselected groups", () => {
        // Select only the ungrouped valve-3
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [{ key: "valve-3", position: { x: 200, y: 200 }, selected: true }],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));

        const after = store.getState()[SLICE_NAME];
        expect(after.copy.nodes).toHaveLength(1);
        expect(after.copy.nodes[0].key).toBe("valve-3");
      });
    });

    describe("cutSelection", () => {
      it("should cut the group node and all its members from the schematic", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.cutSelection({ key: schematicKey }));

        const after = store.getState()[SLICE_NAME];
        const remainingKeys = after.schematics[schematicKey].nodes.map((n) => n.key);
        expect(remainingKeys).not.toContain(groupKey);
        expect(remainingKeys).not.toContain("valve-1");
        expect(remainingKeys).not.toContain("valve-2");
        expect(remainingKeys).toContain("valve-3");
      });

      it("should place group node and members into the copy buffer", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.cutSelection({ key: schematicKey }));

        const after = store.getState()[SLICE_NAME];
        const copyKeys = after.copy.nodes.map((n) => n.key);
        expect(copyKeys).toContain(groupKey);
        expect(copyKeys).toContain("valve-1");
        expect(copyKeys).toContain("valve-2");
      });
    });

    describe("pasteSelection", () => {
      it("should remap groupId references to use the new group key", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({ key: schematicKey, pos: { x: 500, y: 500 } }),
        );

        const after = store.getState()[SLICE_NAME];
        const allProps = after.schematics[schematicKey].props as Record<
          string,
          NodeProps
        >;
        // Find the pasted group node (new key, props.key === "group")
        const pastedNodes = after.schematics[schematicKey].nodes.filter(
          (n) =>
            n.key !== groupKey &&
            n.key !== "valve-1" &&
            n.key !== "valve-2" &&
            n.key !== "valve-3",
        );
        const pastedGroupNode = pastedNodes.find(
          (n) => allProps[n.key]?.key === "group",
        );
        expect(pastedGroupNode).toBeDefined();

        // Pasted members should reference the new group key
        const pastedMembers = pastedNodes.filter(
          (n) => allProps[n.key]?.key !== "group",
        );
        expect(pastedMembers.length).toBe(2);
        for (const member of pastedMembers)
          expect(allProps[member.key].groupId).toBe(pastedGroupNode!.key);
      });

      it("should generate new keys for pasted group nodes and members", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({ key: schematicKey, pos: { x: 500, y: 500 } }),
        );

        const after = store.getState()[SLICE_NAME];
        const allKeys = after.schematics[schematicKey].nodes.map((n) => n.key);
        const uniqueKeys = new Set(allKeys);
        expect(uniqueKeys.size).toBe(allKeys.length);
      });

      it("should select only pasted nodes", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({ key: schematicKey, pos: { x: 500, y: 500 } }),
        );

        const after = store.getState()[SLICE_NAME];
        const originals = after.schematics[schematicKey].nodes.filter(
          (n) =>
            n.key === groupKey ||
            n.key === "valve-1" ||
            n.key === "valve-2" ||
            n.key === "valve-3",
        );
        const pasted = after.schematics[schematicKey].nodes.filter(
          (n) =>
            n.key !== groupKey &&
            n.key !== "valve-1" &&
            n.key !== "valve-2" &&
            n.key !== "valve-3",
        );
        expect(originals.every((n) => !n.selected)).toBe(true);
        expect(pasted.every((n) => n.selected)).toBe(true);
      });
    });

    describe("repeated paste", () => {
      it("should preserve group structure on the second paste", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.copySelection({ key: schematicKey }));

        // First paste
        store.dispatch(
          actions.pasteSelection({ key: schematicKey, pos: { x: 500, y: 500 } }),
        );
        // Second paste
        store.dispatch(
          actions.pasteSelection({ key: schematicKey, pos: { x: 800, y: 800 } }),
        );

        const after = store.getState()[SLICE_NAME];
        const allProps = after.schematics[schematicKey].props as Record<
          string,
          NodeProps
        >;
        // Find both pasted group nodes (exclude the original group)
        const pastedGroupNodes = after.schematics[schematicKey].nodes.filter(
          (n) => n.key !== groupKey && allProps[n.key]?.key === "group",
        );
        expect(pastedGroupNodes).toHaveLength(2);

        // Each pasted group should have exactly 2 members pointing to it
        for (const pg of pastedGroupNodes) {
          const members = after.schematics[schematicKey].nodes.filter(
            (n) => allProps[n.key]?.groupId === pg.key,
          );
          expect(members).toHaveLength(2);
        }
      });
    });

    describe("cut then paste", () => {
      it("should preserve group structure after cut and paste", () => {
        store.dispatch(actions.clearSelection({ key: schematicKey }));
        const state = store.getState()[SLICE_NAME];
        store.dispatch(
          actions.setNodes({
            key: schematicKey,
            nodes: [
              {
                key: groupKey,
                position: state.schematics[schematicKey].nodes.find(
                  (n) => n.key === groupKey,
                )!.position,
                selected: true,
              },
            ],
            mode: "update",
          }),
        );
        store.dispatch(actions.cutSelection({ key: schematicKey }));
        store.dispatch(
          actions.pasteSelection({ key: schematicKey, pos: { x: 500, y: 500 } }),
        );

        const after = store.getState()[SLICE_NAME];
        const allProps = after.schematics[schematicKey].props as Record<
          string,
          NodeProps
        >;
        const pastedGroupNode = after.schematics[schematicKey].nodes.find(
          (n) => allProps[n.key]?.key === "group",
        );
        expect(pastedGroupNode).toBeDefined();

        const pastedMembers = after.schematics[schematicKey].nodes.filter(
          (n) =>
            allProps[n.key]?.key !== "group" &&
            n.key !== "valve-3" &&
            allProps[n.key]?.groupId != null,
        );
        expect(pastedMembers.length).toBe(2);
        for (const m of pastedMembers)
          expect(allProps[m.key].groupId).toBe(pastedGroupNode!.key);
      });
    });
  });

  describe("grouping edge cases", () => {
    const schematicKey = "test-schematic";
    const groupProps: NodeProps = { key: "group" } as NodeProps;

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
    });

    it("should return to ungrouped state after group then ungroup", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));

      const state = store.getState()[SLICE_NAME];
      const groupKey = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      )!.key;

      // Select the group and ungroup
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            {
              key: groupKey,
              position: state.schematics[schematicKey].nodes.find(
                (n) => n.key === groupKey,
              )!.position,
              selected: true,
            },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.ungroupSelection({ key: schematicKey }));

      const after = store.getState()[SLICE_NAME];
      const allProps = after.schematics[schematicKey].props as Record<
        string,
        NodeProps
      >;
      expect(allProps["valve-1"].groupId).toBeUndefined();
      expect(allProps["valve-2"].groupId).toBeUndefined();
      const groupNodes = after.schematics[schematicKey].nodes.filter(
        (n) => allProps[n.key]?.key === "group",
      );
      expect(groupNodes).toHaveLength(0);
      expect(after.schematics[schematicKey].nodes).toHaveLength(2);
    });

    it("should be a no-op when ungrouping with no groups present", () => {
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 0, y: 0 }, selected: true }],
          mode: "update",
        }),
      );
      const before = store.getState()[SLICE_NAME];
      store.dispatch(actions.ungroupSelection({ key: schematicKey }));
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes.length).toBe(
        before.schematics[schematicKey].nodes.length,
      );
    });
  });

  describe("cascade group deletes", () => {
    const schematicKey = "test-schematic";
    const groupProps: NodeProps = { key: "group" } as NodeProps;
    let groupKey: string;

    beforeEach(() => {
      store.dispatch(actions.create({ ...ZERO_STATE, key: schematicKey }));
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-1",
          props: { key: "valve" },
          node: { position: { x: 0, y: 0 } },
        }),
      );
      store.dispatch(
        actions.addElement({
          key: schematicKey,
          elKey: "valve-2",
          props: { key: "valve" },
          node: { position: { x: 100, y: 100 } },
        }),
      );
      // Select both and group them
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [
            { key: "valve-1", position: { x: 0, y: 0 }, selected: true },
            { key: "valve-2", position: { x: 100, y: 100 }, selected: true },
          ],
          mode: "update",
        }),
      );
      store.dispatch(actions.groupSelection({ key: schematicKey, props: groupProps }));
      const state = store.getState()[SLICE_NAME];
      groupKey = state.schematics[schematicKey].nodes.find(
        (n) =>
          (state.schematics[schematicKey].props[n.key] as NodeProps)?.key === "group",
      )!.key;
    });

    it("should remove group children when the group node is deleted", () => {
      const before = store.getState()[SLICE_NAME];
      const nodesWithoutGroup = before.schematics[schematicKey].nodes.filter(
        (n) => n.key !== groupKey,
      );
      // Simulate React Flow removing only the group node
      store.dispatch(actions.setNodes({ key: schematicKey, nodes: nodesWithoutGroup }));
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes).toHaveLength(0);
      const allProps = after.schematics[schematicKey].props;
      expect(allProps["valve-1"]).toBeUndefined();
      expect(allProps["valve-2"]).toBeUndefined();
      expect(allProps[groupKey]).toBeUndefined();
    });

    it("should cascade-delete the entire group when a member is deleted", () => {
      const before = store.getState()[SLICE_NAME];
      const nodesWithoutValve1 = before.schematics[schematicKey].nodes.filter(
        (n) => n.key !== "valve-1",
      );
      store.dispatch(
        actions.setNodes({ key: schematicKey, nodes: nodesWithoutValve1 }),
      );
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes).toHaveLength(0);
      expect(after.schematics[schematicKey].props["valve-1"]).toBeUndefined();
      expect(after.schematics[schematicKey].props["valve-2"]).toBeUndefined();
      expect(after.schematics[schematicKey].props[groupKey]).toBeUndefined();
    });

    it("should not cascade when using update mode", () => {
      // Update mode merges, not replaces — cascade only applies to replace
      store.dispatch(
        actions.setNodes({
          key: schematicKey,
          nodes: [{ key: "valve-1", position: { x: 50, y: 50 } }],
          mode: "update",
        }),
      );
      const after = store.getState()[SLICE_NAME];
      expect(after.schematics[schematicKey].nodes).toHaveLength(3);
    });
  });
});
