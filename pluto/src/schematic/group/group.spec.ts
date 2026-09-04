// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type dimensions, type record } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  buildParentOf,
  canGroup,
  canUngroup,
  createActions,
  fanOutMoves,
  lockedKeys,
  lockMembers,
  remapMembers,
  soleRoot,
  ungroupActions,
  withMembers,
} from "@/schematic/group/group";
import { Node } from "@/schematic/node";

const node = (key: string, x = 0, y = 0): schematic.Node => ({
  key,
  position: { x, y },
  zIndex: 0,
});

const groupConfig = (members: string[]): Node.GroupBox.Config => ({
  variant: Node.GroupBox.VARIANT,
  members,
  dimensions: { width: 100, height: 100 },
});

const doc = (
  nodes: schematic.Node[],
  configs: Record<string, record.Unknown>,
): schematic.Schematic => ({ nodes, configs }) as schematic.Schematic;

// outer holds mid1 (which holds inner1) and mid2, side stands alone, and l1, l2,
// and l3 are loose symbols.
const forest: Record<string, Node.GroupBox.Config> = {
  outer: groupConfig(["mid1", "mid2", "s1"]),
  mid1: groupConfig(["inner1", "s2", "s3"]),
  inner1: groupConfig(["s6", "s7"]),
  mid2: groupConfig(["s4", "s5"]),
  side: groupConfig(["s8", "s9"]),
};
const forestNodes = [
  node("outer"),
  node("mid1", 10, 10),
  node("inner1", 20, 20),
  node("mid2", 30, 30),
  node("side", 40, 40),
  node("s1", 1, 1),
  node("s2", 2, 2),
  node("s3", 3, 3),
  node("s4", 4, 4),
  node("s5", 5, 5),
  node("s6", 6, 6),
  node("s7", 7, 7),
  node("s8", 8, 8),
  node("s9", 9, 9),
  node("l1", 50, 50),
  node("l2", 60, 60),
  node("l3", 70, 70),
];
const forestParentOf = buildParentOf(forest);

describe("group", () => {
  describe("buildParentOf", () => {
    it("should map each member to its group across the forest", () => {
      const parentOf = buildParentOf(forest);
      expect(parentOf.size).toEqual(12);
      expect(parentOf.get("mid1")).toEqual("outer");
      expect(parentOf.get("mid2")).toEqual("outer");
      expect(parentOf.get("inner1")).toEqual("mid1");
      expect(parentOf.get("s6")).toEqual("inner1");
      expect(parentOf.get("s4")).toEqual("mid2");
      expect(parentOf.get("s8")).toEqual("side");
      expect(parentOf.has("outer")).toEqual(false);
      expect(parentOf.has("l1")).toEqual(false);
    });

    it("should ignore non-group configs", () => {
      expect(buildParentOf({ v: { variant: "value" } }).size).toEqual(0);
    });

    it("should keep the first group when two claim the same member", () => {
      const configs = { g1: groupConfig(["a"]), g2: groupConfig(["a"]) };
      expect(buildParentOf(configs).get("a")).toEqual("g1");
    });
  });

  describe("canGroup", () => {
    it("should allow three loose symbols", () => {
      expect(canGroup(["l1", "l2", "l3"], forestNodes, forestParentOf)).toEqual(true);
    });

    it("should reject a single symbol", () => {
      expect(canGroup(["s6"], forestNodes, forestParentOf)).toEqual(false);
    });

    it("should reject a group selected with its members", () => {
      const selected = withMembers(["outer"], forest);
      expect(canGroup(selected, forestNodes, forestParentOf)).toEqual(false);
    });

    it("should reject members that resolve to the same outermost group", () => {
      expect(canGroup(["s6", "s4", "s2", "s1"], forestNodes, forestParentOf)).toEqual(
        false,
      );
    });

    it("should allow deep members of different groups", () => {
      expect(canGroup(["s6", "s8"], forestNodes, forestParentOf)).toEqual(true);
    });

    it("should allow a whole group plus a loose symbol", () => {
      const selected = [...withMembers(["side"], forest), "l1"];
      expect(canGroup(selected, forestNodes, forestParentOf)).toEqual(true);
    });

    it("should ignore keys without nodes", () => {
      expect(canGroup(["s6", "edge1"], forestNodes, forestParentOf)).toEqual(false);
    });
  });

  describe("createActions", () => {
    const flatDims: Record<string, dimensions.Dimensions> = {
      a: { width: 10, height: 10 },
      b: { width: 30, height: 10 },
      c: { width: 20, height: 20 },
      d: { width: 10, height: 40 },
      e: { width: 50, height: 5 },
    };
    const flatNodes = [
      node("a"),
      node("b", 100, 20),
      node("c", -40, 60),
      node("d", 15, -25),
      node("e", 70, 80),
    ];

    it("should insert a group sized to the members' bounding box", () => {
      const result = createActions({
        selected: ["c", "a", "e", "b", "d"],
        nodes: flatNodes,
        configs: {},
        measure: (key) => flatDims[key] ?? null,
      });
      const inserted = result?.actions[0];
      expect(result?.actions).toHaveLength(1);
      expect(inserted).toMatchObject({
        type: "set_node",
        setNode: {
          node: { position: { x: -70, y: -55 }, zIndex: -1 },
          config: {
            variant: "groupBox",
            members: ["c", "a", "e", "b", "d"],
            dimensions: { width: 230, height: 170 },
          },
        },
      });
      if (inserted?.type === "set_node")
        expect(result?.selection).toEqual([
          inserted.setNode.node.key,
          "c",
          "a",
          "e",
          "b",
          "d",
        ]);
    });

    it("should group a member's outermost group, not the member", () => {
      const dims: Record<string, dimensions.Dimensions> = {
        outer: { width: 200, height: 150 },
        l1: { width: 30, height: 30 },
        side: { width: 90, height: 40 },
      };
      const result = createActions({
        selected: ["s6", "l1", "s8"],
        nodes: forestNodes,
        configs: forest,
        measure: (key) => dims[key] ?? null,
      });
      expect(result?.actions[0]).toMatchObject({
        setNode: {
          node: { position: { x: -30, y: -30 } },
          config: {
            members: ["outer", "l1", "side"],
            dimensions: { width: 260, height: 210 },
          },
        },
      });
      expect(result?.selection.slice(1)).toEqual([
        "outer",
        "l1",
        "side",
        "mid1",
        "inner1",
        "s6",
        "s7",
        "s2",
        "s3",
        "mid2",
        "s4",
        "s5",
        "s1",
        "s8",
        "s9",
      ]);
    });

    it("should return null when the selection resolves to one symbol", () => {
      const result = createActions({
        selected: ["l1"],
        nodes: forestNodes,
        configs: forest,
        measure: () => null,
      });
      expect(result).toBeNull();
    });

    it("should return null when the selection holds one group and its members", () => {
      const result = createActions({
        selected: withMembers(["side"], forest),
        nodes: forestNodes,
        configs: forest,
        measure: () => null,
      });
      expect(result).toBeNull();
    });

    it("should return null when a symbol is not measurable", () => {
      const result = createActions({
        selected: ["c", "a", "e", "b", "d"],
        nodes: flatNodes,
        configs: {},
        measure: (key) => (key === "d" ? null : (flatDims[key] ?? null)),
      });
      expect(result).toBeNull();
    });
  });

  describe("remapMembers", () => {
    it("should rewrite members onto pasted keys", () => {
      const config = groupConfig(["m1", "m2", "m3", "m4", "m5"]);
      const remap = { m1: "n1", m2: "n2", m3: "n3", m4: "n4", m5: "n5" };
      expect(remapMembers(config, remap)).toEqual({
        ...config,
        members: ["n1", "n2", "n3", "n4", "n5"],
      });
    });

    it("should drop members that were not pasted", () => {
      const config = groupConfig(["m1", "m2", "m3", "m4", "m5"]);
      expect(remapMembers(config, { m2: "n2", m5: "n5" })).toMatchObject({
        members: ["n2", "n5"],
      });
    });

    it("should pass non-group configs through", () => {
      const config = { variant: "value" };
      expect(remapMembers(config, { a: "a2" })).toBe(config);
      expect(remapMembers(undefined, { a: "a2" })).toBeUndefined();
    });
  });

  describe("withMembers", () => {
    it("should add a selected group's members recursively", () => {
      expect(withMembers(["outer"], forest)).toEqual([
        "outer",
        "mid1",
        "inner1",
        "s6",
        "s7",
        "s2",
        "s3",
        "mid2",
        "s4",
        "s5",
        "s1",
      ]);
    });

    it("should close over each selected key", () => {
      expect(withMembers(["inner1", "l1", "side"], forest)).toEqual([
        "inner1",
        "l1",
        "side",
        "s6",
        "s7",
        "s8",
        "s9",
      ]);
    });

    it("should not duplicate a member already selected", () => {
      expect(withMembers(["s6", "inner1"], forest)).toEqual(["s6", "inner1", "s7"]);
    });

    it("should terminate when corrupt data forms a cycle", () => {
      const configs = { g1: groupConfig(["g2"]), g2: groupConfig(["g1"]) };
      expect(withMembers(["g1"], configs)).toEqual(["g1", "g2"]);
    });
  });

  describe("fanOutMoves", () => {
    it("should apply a moved group's delta to every symbol inside it", () => {
      const actions = fanOutMoves(doc(forestNodes, forest), [
        schematic.setNodePosition({ key: "outer", position: { x: 5, y: -5 } }),
      ]);
      expect(actions.slice(1)).toEqual([
        schematic.setNodePosition({ key: "mid1", position: { x: 15, y: 5 } }),
        schematic.setNodePosition({ key: "inner1", position: { x: 25, y: 15 } }),
        schematic.setNodePosition({ key: "s6", position: { x: 11, y: 1 } }),
        schematic.setNodePosition({ key: "s7", position: { x: 12, y: 2 } }),
        schematic.setNodePosition({ key: "s2", position: { x: 7, y: -3 } }),
        schematic.setNodePosition({ key: "s3", position: { x: 8, y: -2 } }),
        schematic.setNodePosition({ key: "mid2", position: { x: 35, y: 25 } }),
        schematic.setNodePosition({ key: "s4", position: { x: 9, y: -1 } }),
        schematic.setNodePosition({ key: "s5", position: { x: 10, y: 0 } }),
        schematic.setNodePosition({ key: "s1", position: { x: 6, y: -4 } }),
      ]);
    });

    it("should fan out each moved group in a batch", () => {
      const actions = fanOutMoves(doc(forestNodes, forest), [
        schematic.setNodePosition({ key: "mid2", position: { x: 31, y: 31 } }),
        schematic.setNodePosition({ key: "side", position: { x: 42, y: 38 } }),
      ]);
      expect(actions.slice(2)).toEqual([
        schematic.setNodePosition({ key: "s4", position: { x: 5, y: 5 } }),
        schematic.setNodePosition({ key: "s5", position: { x: 6, y: 6 } }),
        schematic.setNodePosition({ key: "s8", position: { x: 10, y: 6 } }),
        schematic.setNodePosition({ key: "s9", position: { x: 11, y: 7 } }),
      ]);
    });

    it("should skip keys already moved in the batch", () => {
      const actions = fanOutMoves(doc(forestNodes, forest), [
        schematic.setNodePosition({ key: "mid1", position: { x: 20, y: 10 } }),
        schematic.setNodePosition({ key: "s2", position: { x: 0, y: 0 } }),
      ]);
      expect(actions).toHaveLength(6);
      expect(actions).not.toContainEqual(
        schematic.setNodePosition({ key: "s2", position: { x: 12, y: 2 } }),
      );
    });

    it("should fan out even when the delta is zero", () => {
      const actions = fanOutMoves(doc(forestNodes, forest), [
        schematic.setNodePosition({ key: "inner1", position: { x: 20, y: 20 } }),
      ]);
      expect(actions.slice(1)).toEqual([
        schematic.setNodePosition({ key: "s6", position: { x: 6, y: 6 } }),
        schematic.setNodePosition({ key: "s7", position: { x: 7, y: 7 } }),
      ]);
    });

    it("should leave moves of non-group symbols untouched", () => {
      const input = [
        schematic.setNodePosition({ key: "l1", position: { x: 1, y: 1 } }),
        schematic.setNodePosition({ key: "s2", position: { x: 2, y: 3 } }),
      ];
      expect(fanOutMoves(doc(forestNodes, forest), input)).toEqual(input);
    });

    it("should return batches without position changes unchanged", () => {
      const input = [schematic.removeNode({ key: "s2" })];
      expect(fanOutMoves(doc(forestNodes, forest), input)).toBe(input);
    });
  });

  describe("lockMembers", () => {
    it("should mark every grouped symbol as non-draggable", () => {
      const locked = lockMembers(forestNodes, forestParentOf);
      locked.forEach((n, i) => {
        if (forestParentOf.has(n.key))
          expect(n).toEqual({ ...forestNodes[i], draggable: false });
        else expect(n).toBe(forestNodes[i]);
      });
    });

    it("should return the input unchanged when there are no groups", () => {
      expect(lockMembers(forestNodes, new Map())).toBe(forestNodes);
    });
  });

  describe("lockedKeys", () => {
    it("should return grouped symbols selected without their group", () => {
      const selected = ["mid2", "s4", "s5", "s6", "l1"];
      expect(lockedKeys(selected, forestParentOf)).toEqual(["mid2", "s6"]);
    });

    it("should pass a selection holding the whole group", () => {
      expect(lockedKeys(["side", "s8", "s9"], forestParentOf)).toEqual([]);
    });

    it("should pass loose symbols", () => {
      expect(lockedKeys(["l1", "l2"], forestParentOf)).toEqual([]);
    });
  });

  describe("soleRoot", () => {
    it("should resolve a group selected with its members", () => {
      expect(soleRoot(withMembers(["outer"], forest), forestParentOf)).toEqual("outer");
    });

    it("should resolve a single loose symbol", () => {
      expect(soleRoot(["l1"], forestParentOf)).toEqual("l1");
    });

    it("should resolve a partial selection inside one group tree", () => {
      expect(soleRoot(["outer", "s6", "mid2"], forestParentOf)).toEqual("outer");
    });

    it("should reject a member selected without its group", () => {
      expect(soleRoot(["s6"], forestParentOf)).toBeNull();
    });

    it("should reject a grouped member beside a loose symbol", () => {
      expect(soleRoot(["s6", "l1"], forestParentOf)).toBeNull();
    });

    it("should reject two unparented keys", () => {
      expect(soleRoot(["l1", "l2"], forestParentOf)).toBeNull();
    });
  });

  describe("canUngroup", () => {
    it("should allow a selection that includes a group", () => {
      expect(canUngroup(["l1", "mid2", "s4"], forest)).toEqual(true);
    });

    it("should reject members selected without their groups", () => {
      expect(canUngroup(["s6", "s7", "s2", "s9"], forest)).toEqual(false);
    });

    it("should reject a selection with no groups", () => {
      expect(canUngroup(["l1", "l2"], forest)).toEqual(false);
    });
  });

  describe("ungroupActions", () => {
    it("should remove a selected group and free its members", () => {
      const result = ungroupActions(withMembers(["side"], forest), forest);
      expect(result?.actions).toEqual([schematic.removeNode({ key: "side" })]);
      expect(result?.freed).toEqual(["s8", "s9"]);
    });

    it("should target a selected member's immediate group", () => {
      const result = ungroupActions(["s6"], forest);
      expect(result?.actions).toEqual([
        schematic.setConfig({
          key: "mid1",
          config: { members: ["s6", "s7", "s2", "s3"] },
        }),
        schematic.removeNode({ key: "inner1" }),
      ]);
      expect(result?.freed).toEqual(["s6", "s7"]);
    });

    it("should remove only the outermost group of a closed selection", () => {
      const result = ungroupActions(withMembers(["outer"], forest), forest);
      expect(result?.actions).toEqual([schematic.removeNode({ key: "outer" })]);
      expect(result?.freed).toEqual([
        "mid1",
        "mid2",
        "s1",
        "inner1",
        "s6",
        "s7",
        "s2",
        "s3",
        "s4",
        "s5",
      ]);
    });

    it("should promote a removed middle group's members into its parent", () => {
      const result = ungroupActions(withMembers(["mid1"], forest), forest);
      expect(result?.actions).toEqual([
        schematic.setConfig({
          key: "outer",
          config: { members: ["inner1", "s2", "s3", "mid2", "s1"] },
        }),
        schematic.removeNode({ key: "mid1" }),
      ]);
      expect(result?.freed).toEqual(["inner1", "s2", "s3", "s6", "s7"]);
    });

    it("should peel a chain one layer per ungroup", () => {
      let configs: Record<string, Node.GroupBox.Config> = { ...forest };
      let selected = withMembers(["outer"], configs);
      const layers: string[][] = [];
      while (layers.length < 3) {
        const result = ungroupActions(selected, configs);
        expect(result).not.toBeNull();
        if (result == null) break;
        const removed = result.actions.flatMap((a) =>
          a.type === "remove_node" ? [a.removeNode.key] : [],
        );
        layers.push(removed);
        configs = Object.fromEntries(
          Object.entries(configs).filter(([key]) => !removed.includes(key)),
        );
        selected = result.freed;
      }
      expect(layers).toEqual([["outer"], ["mid1", "mid2"], ["inner1"]]);
      expect(ungroupActions(selected, configs)).toBeNull();
    });

    it("should return null when the selection touches no group", () => {
      expect(ungroupActions(["l1", "l2"], forest)).toBeNull();
    });
  });

  describe("grouping cycle", () => {
    it("should release members on ungroup and reclaim them on regroup", () => {
      const remaining: Record<string, record.Unknown> = { g: groupConfig(["a", "b"]) };
      expect(buildParentOf(remaining).has("a")).toEqual(true);
      const ungrouped = ungroupActions(["g", "a", "b"], remaining);
      ungrouped?.actions.forEach((a) => {
        if (a.type === "remove_node") delete remaining[a.removeNode.key];
      });
      expect(buildParentOf(remaining).has("a")).toEqual(false);
      const regrouped = createActions({
        selected: ["a", "b"],
        nodes: [node("a"), node("b", 50, 50)],
        configs: remaining,
        measure: () => ({ width: 10, height: 10 }),
      });
      const action = regrouped?.actions[0];
      expect(action?.type).toEqual("set_node");
      if (action?.type !== "set_node") return;
      const { node: groupNode, config } = action.setNode;
      if (config != null) remaining[groupNode.key] = config;
      expect(buildParentOf(remaining).get("a")).toEqual(groupNode.key);
    });

    it("should keep an inner group's members claimed after the outer ungroups", () => {
      const configs = { outer: groupConfig(["inner", "c"]), inner: groupConfig(["a"]) };
      const result = ungroupActions(withMembers(["outer"], configs), configs);
      const remaining: Record<string, record.Unknown> = { ...configs };
      result?.actions.forEach((a) => {
        if (a.type === "remove_node") delete remaining[a.removeNode.key];
      });
      const parentOf = buildParentOf(remaining);
      expect(parentOf.get("a")).toEqual("inner");
      expect(parentOf.has("inner")).toEqual(false);
    });
  });
});
