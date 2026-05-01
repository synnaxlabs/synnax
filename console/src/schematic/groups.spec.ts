// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Diagram } from "@synnaxlabs/pluto";
import { describe, expect, it } from "vitest";

import {
  calculateGroupBoundingBox,
  expandGroupPositions,
  groupKeyOf,
  propagateGroupDrag,
  remapGroupIds,
  resolveAlignmentKey,
  selectedGroupKeys,
} from "@/schematic/groups";
import { type NodeProps } from "@/schematic/types";

const node = (
  key: string,
  position: { x: number; y: number },
  overrides?: Partial<Diagram.Node>,
): Diagram.Node => ({ key, position, selected: false, ...overrides });

const GROUP_PADDING = 30;

describe("groups", () => {
  describe("groupKeyOf", () => {
    it("should return the node key itself when props.key is 'group'", () => {
      expect(groupKeyOf("g1", { key: "group" } as NodeProps)).toBe("g1");
    });

    it("should return the groupId when the node has a groupId", () => {
      expect(groupKeyOf("n1", { key: "valve", groupId: "g1" } as NodeProps)).toBe("g1");
    });

    it("should return undefined when the node is ungrouped", () => {
      expect(groupKeyOf("n1", { key: "valve" } as NodeProps)).toBeUndefined();
    });

    it("should return undefined when groupId is explicitly undefined", () => {
      expect(
        groupKeyOf("n1", { key: "valve", groupId: undefined } as NodeProps),
      ).toBeUndefined();
    });
  });

  describe("propagateGroupDrag", () => {
    it("should return nodes unchanged when no node is dragging", () => {
      const nodes = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const result = propagateGroupDrag(nodes, nodes, {});
      expect(result).toBe(nodes);
    });

    it("should return nodes unchanged when the dragging node is ungrouped", () => {
      const prev = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const curr = [
        node("n1", { x: 10, y: 10 }, { dragging: true } as Partial<Diagram.Node>),
        node("n2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve" } as NodeProps,
        n2: { key: "valve" } as NodeProps,
      };
      const result = propagateGroupDrag(curr, prev, props);
      expect(result).toBe(curr);
    });

    it("should return nodes unchanged when the delta is zero", () => {
      const nodes = [
        node("g1", { x: 0, y: 0 }, { dragging: true } as Partial<Diagram.Node>),
        node("n1", { x: 50, y: 50 }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        n1: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const result = propagateGroupDrag(nodes, nodes, props);
      expect(result).toBe(nodes);
    });

    it("should apply drag delta to all non-dragging members of the same group", () => {
      const prev = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }),
        node("n2", { x: 100, y: 100 }),
      ];
      const curr = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 60, y: 70 }, { dragging: true } as Partial<Diagram.Node>),
        node("n2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        n1: { key: "valve", groupId: "g1" } as NodeProps,
        n2: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const result = propagateGroupDrag(curr, prev, props);
      expect(result[0].position).toEqual({ x: 10, y: 20 });
      expect(result[1].position).toEqual({ x: 60, y: 70 });
      expect(result[2].position).toEqual({ x: 110, y: 120 });
    });

    it("should not move nodes belonging to a different group", () => {
      const prev = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const curr = [
        node("n1", { x: 10, y: 10 }, { dragging: true } as Partial<Diagram.Node>),
        node("n2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve", groupId: "gA" } as NodeProps,
        n2: { key: "valve", groupId: "gB" } as NodeProps,
      };
      const result = propagateGroupDrag(curr, prev, props);
      expect(result[1].position).toEqual({ x: 100, y: 100 });
    });

    it("should not move ungrouped nodes", () => {
      const prev = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const curr = [
        node("n1", { x: 10, y: 10 }, { dragging: true } as Partial<Diagram.Node>),
        node("n2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve", groupId: "g1" } as NodeProps,
        n2: { key: "valve" } as NodeProps,
      };
      const result = propagateGroupDrag(curr, prev, props);
      expect(result[1].position).toEqual({ x: 100, y: 100 });
    });

    it("should move the group box node when a member is dragged", () => {
      const prev = [node("g1", { x: 0, y: 0 }), node("n1", { x: 50, y: 50 })];
      const curr = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 70, y: 80 }, { dragging: true } as Partial<Diagram.Node>),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        n1: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const result = propagateGroupDrag(curr, prev, props);
      expect(result[0].position).toEqual({ x: 20, y: 30 });
    });

    it("should return nodes unchanged when prevNodes doesn't contain the dragging node", () => {
      const curr = [
        node("n1", { x: 10, y: 10 }, { dragging: true } as Partial<Diagram.Node>),
        node("n2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const result = propagateGroupDrag(curr, [], props);
      expect(result).toBe(curr);
    });
  });

  describe("remapGroupIds", () => {
    it("should remap groupId references using the keyMap", () => {
      const props: Record<string, NodeProps> = {
        "new-n1": { key: "valve", groupId: "old-g1" } as NodeProps,
      };
      remapGroupIds(props, { "old-g1": "new-g1", "old-n1": "new-n1" });
      expect(props["new-n1"].groupId).toBe("new-g1");
    });

    it("should not modify props whose groupId is not in the keyMap", () => {
      const props: Record<string, NodeProps> = {
        "new-n1": { key: "valve", groupId: "unknown" } as NodeProps,
      };
      remapGroupIds(props, { "old-n1": "new-n1" });
      expect(props["new-n1"].groupId).toBe("unknown");
    });

    it("should not modify props without a groupId", () => {
      const props: Record<string, NodeProps> = {
        "new-n1": { key: "valve" } as NodeProps,
      };
      remapGroupIds(props, { "old-n1": "new-n1" });
      expect(props["new-n1"].groupId).toBeUndefined();
    });

    it("should handle multiple remappings correctly", () => {
      const props: Record<string, NodeProps> = {
        "new-n1": { key: "valve", groupId: "old-g1" } as NodeProps,
        "new-n2": { key: "valve", groupId: "old-g1" } as NodeProps,
      };
      remapGroupIds(props, {
        "old-g1": "new-g1",
        "old-n1": "new-n1",
        "old-n2": "new-n2",
      });
      expect(props["new-n1"].groupId).toBe("new-g1");
      expect(props["new-n2"].groupId).toBe("new-g1");
    });
  });

  describe("resolveAlignmentKey", () => {
    it("should return the group node key and position for a grouped member", () => {
      const nodes = [node("g1", { x: 10, y: 20 }), node("n1", { x: 50, y: 60 })];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        n1: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const result = resolveAlignmentKey("n1", props, nodes, { x: 50, y: 60 });
      expect(result).toEqual({ key: "g1", position: { x: 10, y: 20 } });
    });

    it("should return the element's own key and position when ungrouped", () => {
      const nodes = [node("n1", { x: 50, y: 60 })];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve" } as NodeProps,
      };
      const result = resolveAlignmentKey("n1", props, nodes, { x: 50, y: 60 });
      expect(result).toEqual({ key: "n1", position: { x: 50, y: 60 } });
    });

    it("should return the element's own key when the element IS the group node", () => {
      const nodes = [node("g1", { x: 10, y: 20 })];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
      };
      const result = resolveAlignmentKey("g1", props, nodes, { x: 10, y: 20 });
      expect(result).toEqual({ key: "g1", position: { x: 10, y: 20 } });
    });

    it("should return the element's own key when the group node is not in the nodes array", () => {
      const nodes = [node("n1", { x: 50, y: 60 })];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const result = resolveAlignmentKey("n1", props, nodes, { x: 50, y: 60 });
      expect(result).toEqual({ key: "n1", position: { x: 50, y: 60 } });
    });

    it("should return the element's own key when props is missing for the element", () => {
      const nodes = [node("n1", { x: 50, y: 60 })];
      const result = resolveAlignmentKey("n1", {}, nodes, { x: 50, y: 60 });
      expect(result).toEqual({ key: "n1", position: { x: 50, y: 60 } });
    });
  });

  describe("expandGroupPositions", () => {
    it("should expand a group position change to all member nodes", () => {
      const nodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }),
        node("n2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        n1: { key: "valve", groupId: "g1" } as NodeProps,
        n2: { key: "valve", groupId: "g1" } as NodeProps,
      };
      const positions: [string, { x: number; y: number }][] = [
        ["g1", { x: 10, y: 20 }],
      ];
      const result = expandGroupPositions(positions, nodes, props);
      expect(result).toEqual([
        ["g1", { x: 10, y: 20 }],
        ["n1", { x: 60, y: 70 }],
        ["n2", { x: 110, y: 120 }],
      ]);
    });

    it("should pass through non-group node positions unchanged", () => {
      const nodes = [node("n1", { x: 50, y: 50 })];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve" } as NodeProps,
      };
      const positions: [string, { x: number; y: number }][] = [
        ["n1", { x: 100, y: 200 }],
      ];
      const result = expandGroupPositions(positions, nodes, props);
      expect(result).toEqual([["n1", { x: 100, y: 200 }]]);
    });

    it("should skip a group position entry if the group node is not found", () => {
      const nodes = [node("n1", { x: 50, y: 50 })];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
      };
      const positions: [string, { x: number; y: number }][] = [
        ["g1", { x: 10, y: 20 }],
      ];
      const result = expandGroupPositions(positions, nodes, props);
      expect(result).toEqual([]);
    });

    it("should handle mixed group and non-group position entries", () => {
      const nodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }),
        node("n2", { x: 200, y: 200 }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        n1: { key: "valve", groupId: "g1" } as NodeProps,
        n2: { key: "valve" } as NodeProps,
      };
      const positions: [string, { x: number; y: number }][] = [
        ["g1", { x: 10, y: 10 }],
        ["n2", { x: 300, y: 300 }],
      ];
      const result = expandGroupPositions(positions, nodes, props);
      expect(result).toEqual([
        ["g1", { x: 10, y: 10 }],
        ["n1", { x: 60, y: 60 }],
        ["n2", { x: 300, y: 300 }],
      ]);
    });

    it("should handle an empty positions array", () => {
      const result = expandGroupPositions([], [], {});
      expect(result).toEqual([]);
    });

    it("should handle a group with no members", () => {
      const nodes = [node("g1", { x: 0, y: 0 })];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
      };
      const positions: [string, { x: number; y: number }][] = [
        ["g1", { x: 10, y: 20 }],
      ];
      const result = expandGroupPositions(positions, nodes, props);
      expect(result).toEqual([["g1", { x: 10, y: 20 }]]);
    });
  });

  describe("calculateGroupBoundingBox", () => {
    it("should compute bounding box with padding for two nodes", () => {
      const members = [
        node("n1", { x: 100, y: 100 }, { measured: { width: 50, height: 30 } }),
        node("n2", { x: 200, y: 200 }, { measured: { width: 60, height: 40 } }),
      ];
      const result = calculateGroupBoundingBox(members);
      expect(result.position).toEqual({
        x: 100 - GROUP_PADDING,
        y: 100 - GROUP_PADDING,
      });
      expect(result.dimensions).toEqual({
        width: 200 + 60 - 100 + 2 * GROUP_PADDING,
        height: 200 + 40 - 100 + 2 * GROUP_PADDING,
      });
    });

    it("should default to 0 when measured dimensions are missing", () => {
      const members = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const result = calculateGroupBoundingBox(members);
      expect(result.position).toEqual({ x: -GROUP_PADDING, y: -GROUP_PADDING });
      expect(result.dimensions).toEqual({
        width: 100 + 2 * GROUP_PADDING,
        height: 100 + 2 * GROUP_PADDING,
      });
    });

    it("should handle a single node", () => {
      const members = [
        node("n1", { x: 50, y: 50 }, { measured: { width: 80, height: 40 } }),
      ];
      const result = calculateGroupBoundingBox(members);
      expect(result.position).toEqual({
        x: 50 - GROUP_PADDING,
        y: 50 - GROUP_PADDING,
      });
      expect(result.dimensions).toEqual({
        width: 80 + 2 * GROUP_PADDING,
        height: 40 + 2 * GROUP_PADDING,
      });
    });

    it("should handle nodes at negative coordinates", () => {
      const members = [
        node("n1", { x: -100, y: -50 }, { measured: { width: 20, height: 20 } }),
        node("n2", { x: 50, y: 100 }, { measured: { width: 20, height: 20 } }),
      ];
      const result = calculateGroupBoundingBox(members);
      expect(result.position).toEqual({
        x: -100 - GROUP_PADDING,
        y: -50 - GROUP_PADDING,
      });
      expect(result.dimensions).toEqual({
        width: 50 + 20 - -100 + 2 * GROUP_PADDING,
        height: 100 + 20 - -50 + 2 * GROUP_PADDING,
      });
    });

    it("should handle all nodes at the same position", () => {
      const members = [
        node("n1", { x: 100, y: 100 }, { measured: { width: 40, height: 40 } }),
        node("n2", { x: 100, y: 100 }, { measured: { width: 40, height: 40 } }),
      ];
      const result = calculateGroupBoundingBox(members);
      expect(result.position).toEqual({
        x: 100 - GROUP_PADDING,
        y: 100 - GROUP_PADDING,
      });
      expect(result.dimensions).toEqual({
        width: 40 + 2 * GROUP_PADDING,
        height: 40 + 2 * GROUP_PADDING,
      });
    });
  });

  describe("selectedGroupKeys", () => {
    it("should return keys of selected group nodes", () => {
      const nodes = [
        node("g1", { x: 0, y: 0 }, { selected: true }),
        node("g2", { x: 100, y: 100 }, { selected: true }),
        node("n1", { x: 50, y: 50 }, { selected: true }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        g2: { key: "group" } as NodeProps,
        n1: { key: "valve" } as NodeProps,
      };
      const result = selectedGroupKeys(nodes, props);
      expect(result).toEqual(new Set(["g1", "g2"]));
    });

    it("should exclude unselected group nodes", () => {
      const nodes = [
        node("g1", { x: 0, y: 0 }, { selected: true }),
        node("g2", { x: 100, y: 100 }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
        g2: { key: "group" } as NodeProps,
      };
      const result = selectedGroupKeys(nodes, props);
      expect(result).toEqual(new Set(["g1"]));
    });

    it("should return an empty set when no groups are selected", () => {
      const nodes = [
        node("n1", { x: 0, y: 0 }, { selected: true }),
        node("n2", { x: 100, y: 100 }, { selected: true }),
      ];
      const props: Record<string, NodeProps> = {
        n1: { key: "valve" } as NodeProps,
        n2: { key: "valve" } as NodeProps,
      };
      const result = selectedGroupKeys(nodes, props);
      expect(result).toEqual(new Set());
    });

    it("should return an empty set when nodes array is empty", () => {
      const result = selectedGroupKeys([], {});
      expect(result).toEqual(new Set());
    });

    it("should handle nodes with missing props", () => {
      const nodes = [
        node("g1", { x: 0, y: 0 }, { selected: true }),
        node("n1", { x: 50, y: 50 }, { selected: true }),
      ];
      const props: Record<string, NodeProps> = {
        g1: { key: "group" } as NodeProps,
      };
      const result = selectedGroupKeys(nodes, props);
      expect(result).toEqual(new Set(["g1"]));
    });
  });
});
