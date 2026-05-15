// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type record, type xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  auditGroups,
  cascadeRemovedKeys,
  computeGroupBoundingBox,
  expandGroupPositions,
  expandSelectionToGroups,
  findGroupMembers,
  findOrphanedGroups,
  GROUP_PADDING,
  GROUP_VARIANT,
  groupKeyOf,
  isGroupContainer,
  propagateGroupDrag,
  remapGroupId,
  resolveAlignmentKey,
  selectedGroupKeys,
} from "@/schematic/groups";

const node = (
  key: string,
  position: xy.XY,
  extras?: Partial<schematic.Node>,
): schematic.Node => ({ key, position, ...extras });

const groupCfg: record.Unknown = { variant: GROUP_VARIANT };
const valveCfg: record.Unknown = { variant: "tank" };

describe("groups", () => {
  describe("groupKeyOf", () => {
    it("should return the node key itself when the node is a group container", () => {
      expect(groupKeyOf(node("g1", { x: 0, y: 0 }), { g1: groupCfg })).toBe("g1");
    });

    it("should return the groupId when the node has a groupId", () => {
      expect(
        groupKeyOf(node("n1", { x: 0, y: 0 }, { groupId: "g1" }), { n1: valveCfg }),
      ).toBe("g1");
    });

    it("should return undefined when the node is ungrouped", () => {
      expect(groupKeyOf(node("n1", { x: 0, y: 0 }), { n1: valveCfg })).toBeUndefined();
    });

    it("should return undefined when groupId is explicitly undefined", () => {
      expect(
        groupKeyOf(node("n1", { x: 0, y: 0 }, { groupId: undefined }), {
          n1: valveCfg,
        }),
      ).toBeUndefined();
    });
  });

  describe("propagateGroupDrag", () => {
    it("should return an empty array when no changes are passed", () => {
      const allNodes = [node("n1", { x: 0, y: 0 })];
      expect(propagateGroupDrag([], allNodes, { n1: valveCfg })).toEqual([]);
    });

    it("should return changes unchanged when the dragging node is ungrouped", () => {
      const allNodes = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const changes = [{ key: "n1", position: { x: 10, y: 10 } }];
      const result = propagateGroupDrag(changes, allNodes, {
        n1: valveCfg,
        n2: valveCfg,
      });
      expect(result).toEqual(changes);
    });

    it("should return changes unchanged when the delta is zero", () => {
      const allNodes = [node("g1", { x: 0, y: 0 }), node("n1", { x: 50, y: 50 })];
      const changes = [{ key: "g1", position: { x: 0, y: 0 } }];
      const result = propagateGroupDrag(changes, allNodes, {
        g1: groupCfg,
        n1: valveCfg,
      });
      expect(result).toEqual(changes);
    });

    it("should apply drag delta to all non-dragging members of the same group", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 100, y: 100 }, { groupId: "g1" }),
      ];
      const changes = [{ key: "n1", position: { x: 60, y: 70 } }];
      const result = propagateGroupDrag(changes, allNodes, {
        g1: groupCfg,
        n1: valveCfg,
        n2: valveCfg,
      });
      expect(result).toEqual(
        expect.arrayContaining([
          { key: "n1", position: { x: 60, y: 70 } },
          { key: "g1", position: { x: 10, y: 20 } },
          { key: "n2", position: { x: 110, y: 120 } },
        ]),
      );
      expect(result).toHaveLength(3);
    });

    it("should not move nodes belonging to a different group", () => {
      const allNodes = [
        node("gA", { x: 0, y: 0 }),
        node("gB", { x: 200, y: 200 }),
        node("n1", { x: 50, y: 50 }, { groupId: "gA" }),
        node("n2", { x: 250, y: 250 }, { groupId: "gB" }),
      ];
      const changes = [{ key: "n1", position: { x: 60, y: 60 } }];
      const result = propagateGroupDrag(changes, allNodes, {
        gA: groupCfg,
        gB: groupCfg,
        n1: valveCfg,
        n2: valveCfg,
      });
      expect(result.find((c) => c.key === "n2")).toBeUndefined();
      expect(result.find((c) => c.key === "gB")).toBeUndefined();
    });

    it("should not move ungrouped nodes", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 200, y: 200 }),
      ];
      const changes = [{ key: "n1", position: { x: 60, y: 60 } }];
      const result = propagateGroupDrag(changes, allNodes, {
        g1: groupCfg,
        n1: valveCfg,
        n2: valveCfg,
      });
      expect(result.find((c) => c.key === "n2")).toBeUndefined();
    });

    it("should move the group container when a member is dragged", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
      ];
      const changes = [{ key: "n1", position: { x: 70, y: 80 } }];
      const result = propagateGroupDrag(changes, allNodes, {
        g1: groupCfg,
        n1: valveCfg,
      });
      const g1Change = result.find((c) => c.key === "g1");
      expect(g1Change?.position).toEqual({ x: 20, y: 30 });
    });

    it("should return changes unchanged when allNodes doesn't contain the dragging node", () => {
      const changes = [{ key: "n1", position: { x: 10, y: 10 } }];
      const result = propagateGroupDrag(changes, [], { n1: valveCfg });
      expect(result).toEqual(changes);
    });
  });

  describe("remapGroupId", () => {
    it("should remap a node's groupId via the keyMap", () => {
      const n = node("new-n1", { x: 0, y: 0 }, { groupId: "old-g1" });
      expect(remapGroupId(n, { "old-g1": "new-g1" })).toEqual({
        ...n,
        groupId: "new-g1",
      });
    });

    it("should drop groupId when the container is not in the keyMap (dangling-ref fix)", () => {
      const n = node("new-n1", { x: 0, y: 0 }, { groupId: "unknown" });
      const result = remapGroupId(n, { "old-n1": "new-n1" });
      expect(result.groupId).toBeUndefined();
      expect(result.key).toBe("new-n1");
    });

    it("should return a node without a groupId unchanged", () => {
      const n = node("new-n1", { x: 0, y: 0 });
      expect(remapGroupId(n, { "old-n1": "new-n1" })).toEqual(n);
    });

    it("should handle multiple remappings when applied per node", () => {
      const n1 = node("new-n1", { x: 0, y: 0 }, { groupId: "old-g1" });
      const n2 = node("new-n2", { x: 50, y: 50 }, { groupId: "old-g1" });
      const keyMap = { "old-g1": "new-g1" };
      expect(remapGroupId(n1, keyMap).groupId).toBe("new-g1");
      expect(remapGroupId(n2, keyMap).groupId).toBe("new-g1");
    });
  });

  describe("resolveAlignmentKey", () => {
    it("should return the group container's key and position for a grouped member", () => {
      const allNodes = [
        node("g1", { x: 10, y: 20 }),
        node("n1", { x: 50, y: 60 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg };
      expect(resolveAlignmentKey("n1", allNodes, configs, { x: 50, y: 60 })).toEqual({
        key: "g1",
        position: { x: 10, y: 20 },
      });
    });

    it("should return the element's own key and position when ungrouped", () => {
      const allNodes = [node("n1", { x: 50, y: 60 })];
      expect(
        resolveAlignmentKey("n1", allNodes, { n1: valveCfg }, { x: 50, y: 60 }),
      ).toEqual({ key: "n1", position: { x: 50, y: 60 } });
    });

    it("should return the element's own key when the element IS the group container", () => {
      const allNodes = [node("g1", { x: 10, y: 20 })];
      expect(
        resolveAlignmentKey("g1", allNodes, { g1: groupCfg }, { x: 10, y: 20 }),
      ).toEqual({ key: "g1", position: { x: 10, y: 20 } });
    });

    it("should return the element's own key when the group container is not in the nodes array", () => {
      const allNodes = [node("n1", { x: 50, y: 60 }, { groupId: "g1" })];
      expect(
        resolveAlignmentKey("n1", allNodes, { n1: valveCfg }, { x: 50, y: 60 }),
      ).toEqual({ key: "n1", position: { x: 50, y: 60 } });
    });

    it("should return the element's own key when configs has no entry for it", () => {
      const allNodes = [node("n1", { x: 50, y: 60 })];
      expect(resolveAlignmentKey("n1", allNodes, {}, { x: 50, y: 60 })).toEqual({
        key: "n1",
        position: { x: 50, y: 60 },
      });
    });
  });

  describe("expandGroupPositions", () => {
    it("should expand a group position change to all member nodes", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 100, y: 100 }, { groupId: "g1" }),
      ];
      const result = expandGroupPositions([["g1", { x: 10, y: 20 }]], allNodes, {
        g1: groupCfg,
        n1: valveCfg,
        n2: valveCfg,
      });
      expect(result).toEqual([
        ["g1", { x: 10, y: 20 }],
        ["n1", { x: 60, y: 70 }],
        ["n2", { x: 110, y: 120 }],
      ]);
    });

    it("should pass through non-group node positions unchanged", () => {
      const allNodes = [node("n1", { x: 50, y: 50 })];
      const result = expandGroupPositions([["n1", { x: 100, y: 200 }]], allNodes, {
        n1: valveCfg,
      });
      expect(result).toEqual([["n1", { x: 100, y: 200 }]]);
    });

    it("should skip a group position entry if the group container is not found", () => {
      const allNodes = [node("n1", { x: 50, y: 50 })];
      const result = expandGroupPositions([["g1", { x: 10, y: 20 }]], allNodes, {
        g1: groupCfg,
      });
      expect(result).toEqual([]);
    });

    it("should handle mixed group and non-group position entries", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 200, y: 200 }),
      ];
      const result = expandGroupPositions(
        [
          ["g1", { x: 10, y: 10 }],
          ["n2", { x: 300, y: 300 }],
        ],
        allNodes,
        { g1: groupCfg, n1: valveCfg, n2: valveCfg },
      );
      expect(result).toEqual([
        ["g1", { x: 10, y: 10 }],
        ["n1", { x: 60, y: 60 }],
        ["n2", { x: 300, y: 300 }],
      ]);
    });

    it("should handle an empty positions array", () => {
      expect(expandGroupPositions([], [], {})).toEqual([]);
    });

    it("should handle a group with no members", () => {
      const allNodes = [node("g1", { x: 0, y: 0 })];
      const result = expandGroupPositions([["g1", { x: 10, y: 20 }]], allNodes, {
        g1: groupCfg,
      });
      expect(result).toEqual([["g1", { x: 10, y: 20 }]]);
    });
  });

  describe("computeGroupBoundingBox", () => {
    it("should compute bounding box with padding for two nodes", () => {
      const members = [
        node("n1", { x: 100, y: 100 }, { measured: { width: 50, height: 30 } }),
        node("n2", { x: 200, y: 200 }, { measured: { width: 60, height: 40 } }),
      ];
      const result = computeGroupBoundingBox(members);
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
      const result = computeGroupBoundingBox(members);
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
      const result = computeGroupBoundingBox(members);
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
      const result = computeGroupBoundingBox(members);
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
      const result = computeGroupBoundingBox(members);
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
    it("should return keys of selected group containers", () => {
      const configs = { g1: groupCfg, g2: groupCfg, n1: valveCfg };
      expect(selectedGroupKeys(["g1", "g2", "n1"], configs)).toEqual(
        new Set(["g1", "g2"]),
      );
    });

    it("should exclude non-group selected keys", () => {
      const configs = { g1: groupCfg, n1: valveCfg };
      expect(selectedGroupKeys(["g1", "n1"], configs)).toEqual(new Set(["g1"]));
    });

    it("should return an empty set when no group containers are selected", () => {
      expect(selectedGroupKeys(["n1", "n2"], { n1: valveCfg, n2: valveCfg })).toEqual(
        new Set(),
      );
    });

    it("should return an empty set when the selection is empty", () => {
      expect(selectedGroupKeys([], {})).toEqual(new Set());
    });

    it("should handle keys with missing config entries", () => {
      expect(selectedGroupKeys(["g1", "n1"], { g1: groupCfg })).toEqual(
        new Set(["g1"]),
      );
    });
  });

  describe("isGroupContainer", () => {
    it("should return true when the config's variant is the group variant", () => {
      expect(isGroupContainer("g1", { g1: groupCfg })).toBe(true);
    });

    it("should return false when the config's variant is not the group variant", () => {
      expect(isGroupContainer("n1", { n1: valveCfg })).toBe(false);
    });

    it("should return false when the key has no entry in configs", () => {
      expect(isGroupContainer("missing", {})).toBe(false);
    });
  });

  describe("expandSelectionToGroups", () => {
    it("should return the selection unchanged when no selected key touches a group", () => {
      const allNodes = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const configs = { n1: valveCfg, n2: valveCfg };
      expect(expandSelectionToGroups(["n1"], allNodes, configs).sort()).toEqual(["n1"]);
    });

    it("should pull in all members when a group container is selected", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 100, y: 100 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg, n2: valveCfg };
      expect(expandSelectionToGroups(["g1"], allNodes, configs).sort()).toEqual([
        "g1",
        "n1",
        "n2",
      ]);
    });

    it("should pull in the container when a member is selected", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 100, y: 100 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg, n2: valveCfg };
      expect(expandSelectionToGroups(["n1"], allNodes, configs).sort()).toEqual([
        "g1",
        "n1",
        "n2",
      ]);
    });

    it("should be idempotent when the selection already includes the full group", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg };
      expect(expandSelectionToGroups(["g1", "n1"], allNodes, configs).sort()).toEqual([
        "g1",
        "n1",
      ]);
    });
  });

  describe("findGroupMembers", () => {
    it("should return the keys of nodes whose groupId matches", () => {
      const allNodes = [
        node("n1", { x: 0, y: 0 }, { groupId: "g1" }),
        node("n2", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n3", { x: 100, y: 100 }, { groupId: "g2" }),
        node("n4", { x: 200, y: 200 }),
      ];
      expect(findGroupMembers("g1", allNodes).sort()).toEqual(["n1", "n2"]);
    });

    it("should return an empty list when no nodes match the group key", () => {
      expect(findGroupMembers("g1", [node("n1", { x: 0, y: 0 })])).toEqual([]);
    });
  });

  describe("findOrphanedGroups", () => {
    it("should report group containers with no members", () => {
      const allNodes = [node("g1", { x: 0, y: 0 })];
      const configs = { g1: groupCfg };
      expect(findOrphanedGroups(allNodes, configs)).toEqual(["g1"]);
    });

    it("should not report group containers that still have members", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg };
      expect(findOrphanedGroups(allNodes, configs)).toEqual([]);
    });

    it("should use the minMembers threshold to flag under-populated groups", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg };
      expect(findOrphanedGroups(allNodes, configs, 2)).toEqual(["g1"]);
    });
  });

  describe("cascadeRemovedKeys", () => {
    it("should cascade-remove members when a group container is removed", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 100, y: 100 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg, n2: valveCfg };
      expect(cascadeRemovedKeys(["g1"], allNodes, configs).sort()).toEqual([
        "g1",
        "n1",
        "n2",
      ]);
    });

    it("should cascade-remove the whole group when a single member is removed (atomic groups)", () => {
      const allNodes = [
        node("g1", { x: 0, y: 0 }),
        node("n1", { x: 50, y: 50 }, { groupId: "g1" }),
        node("n2", { x: 100, y: 100 }, { groupId: "g1" }),
      ];
      const configs = { g1: groupCfg, n1: valveCfg, n2: valveCfg };
      expect(cascadeRemovedKeys(["n1"], allNodes, configs).sort()).toEqual([
        "g1",
        "n1",
        "n2",
      ]);
    });

    it("should leave unrelated nodes alone when a non-group node is removed", () => {
      const allNodes = [node("n1", { x: 0, y: 0 }), node("n2", { x: 100, y: 100 })];
      const configs = { n1: valveCfg, n2: valveCfg };
      expect(cascadeRemovedKeys(["n1"], allNodes, configs).sort()).toEqual(["n1"]);
    });
  });

  describe("auditGroups", () => {
    it("should flag a group container with zero members for removal", () => {
      const allNodes = [node("g1", { x: 0, y: 0 })];
      const configs = { g1: { ...groupCfg, dimensions: { width: 100, height: 100 } } };
      const audit = auditGroups(allNodes, configs);
      expect(audit.removeGroupKeys).toEqual(["g1"]);
      expect(audit.clearGroupIdNodes).toEqual([]);
      expect(audit.resizeGroups).toEqual([]);
    });

    it("should flag a single-member group for removal and clear the survivor's groupId", () => {
      const survivor = node("n1", { x: 50, y: 50 }, { groupId: "g1" });
      const allNodes = [node("g1", { x: 0, y: 0 }), survivor];
      const configs = {
        g1: { ...groupCfg, dimensions: { width: 100, height: 100 } },
        n1: valveCfg,
      };
      const audit = auditGroups(allNodes, configs);
      expect(audit.removeGroupKeys).toEqual(["g1"]);
      expect(audit.clearGroupIdNodes).toHaveLength(1);
      expect(audit.clearGroupIdNodes[0].key).toBe("n1");
      expect(audit.clearGroupIdNodes[0].groupId).toBeUndefined();
    });

    it("should resize a multi-member group whose bounding box has changed", () => {
      const allNodes = [
        node("g1", { x: 100, y: 100 }),
        node(
          "n1",
          { x: 0, y: 0 },
          {
            measured: { width: 50, height: 50 },
            groupId: "g1",
          },
        ),
        node(
          "n2",
          { x: 100, y: 100 },
          {
            measured: { width: 50, height: 50 },
            groupId: "g1",
          },
        ),
      ];
      const configs = {
        g1: { ...groupCfg, dimensions: { width: 0, height: 0 } },
        n1: valveCfg,
        n2: valveCfg,
      };
      const audit = auditGroups(allNodes, configs);
      expect(audit.removeGroupKeys).toEqual([]);
      expect(audit.resizeGroups).toHaveLength(1);
      expect(audit.resizeGroups[0].key).toBe("g1");
      expect(audit.resizeGroups[0].position).toEqual({
        x: -GROUP_PADDING,
        y: -GROUP_PADDING,
      });
    });

    it("should not resize a multi-member group whose bounding box is already correct", () => {
      const allNodes = [
        node("g1", { x: -GROUP_PADDING, y: -GROUP_PADDING }),
        node(
          "n1",
          { x: 0, y: 0 },
          {
            measured: { width: 50, height: 50 },
            groupId: "g1",
          },
        ),
        node(
          "n2",
          { x: 100, y: 100 },
          {
            measured: { width: 50, height: 50 },
            groupId: "g1",
          },
        ),
      ];
      const configs = {
        g1: {
          ...groupCfg,
          dimensions: {
            width: 150 + 2 * GROUP_PADDING,
            height: 150 + 2 * GROUP_PADDING,
          },
        },
        n1: valveCfg,
        n2: valveCfg,
      };
      const audit = auditGroups(allNodes, configs);
      expect(audit.removeGroupKeys).toEqual([]);
      expect(audit.resizeGroups).toEqual([]);
    });
  });
});
