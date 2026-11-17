// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { assert, describe, expect, it } from "vitest";

import {
  buildParentOf,
  canGroup,
  canUngroup,
  childrenOf,
  closure,
  createActions,
  drillIn,
  fanOutMoves,
  lockMembers,
  shielded,
  ungroupActions,
  withMembers,
} from "@/schematic/group/group";
import { Node } from "@/schematic/node";

const vent = schematic.nodeConfigZ.parse({ variant: "vent" });

const groupConfig = (): Node.GroupBox.Config => ({
  variant: Node.GroupBox.VARIANT,
  locked: false,
});

const node = (
  key: string,
  x = 0,
  y = 0,
  parent?: string,
  config: schematic.NodeConfig = vent,
): schematic.Node => ({
  key,
  position: { x, y },
  zIndex: 0,
  config,
  ...(parent != null && { parent }),
});

const group = (key: string, x = 0, y = 0, parent?: string): schematic.Node =>
  node(key, x, y, parent, groupConfig());

const doc = (nodes: schematic.Node[]): schematic.Schematic =>
  schematic.schematicZ.parse({ name: "doc", nodes });

const configsByKey = (
  nodes: readonly schematic.Node[],
): Record<string, schematic.ElementConfig> =>
  Object.fromEntries(nodes.map((n) => [n.key, n.config]));

// outer holds mid1 (which holds inner1) and mid2, side stands alone, and l1, l2,
// and l3 are loose symbols.
const forestNodes = [
  group("outer"),
  group("mid1", 10, 10, "outer"),
  group("inner1", 20, 20, "mid1"),
  group("mid2", 30, 30, "outer"),
  group("side", 40, 40),
  node("s1", 1, 1, "outer"),
  node("s2", 2, 2, "mid1"),
  node("s3", 3, 3, "mid1"),
  node("s4", 4, 4, "mid2"),
  node("s5", 5, 5, "mid2"),
  node("s6", 6, 6, "inner1"),
  node("s7", 7, 7, "inner1"),
  node("s8", 8, 8, "side"),
  node("s9", 9, 9, "side"),
  node("l1", 50, 50),
  node("l2", 60, 60),
  node("l3", 70, 70),
];
const forest = configsByKey(forestNodes);
const forestParentOf = buildParentOf(forestNodes);

const withLocked = (key: string): Record<string, schematic.ElementConfig> => ({
  ...forest,
  [key]: { ...groupConfig(), locked: true },
});

// Applies the actions with the client reducer, as a dispatch does.
const reduce = (
  nodes: schematic.Node[],
  actions: schematic.Action[],
): schematic.Node[] => schematic.reduceAll(doc(nodes), actions).next.nodes;

describe("group", () => {
  describe("buildParentOf", () => {
    it("should map each grouped node to its group across the forest", () => {
      const parentOf = buildParentOf(forestNodes);
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
  });

  describe("childrenOf", () => {
    it("should list the children of each group in node order", () => {
      const children = childrenOf(forestParentOf);
      expect(children.get("outer")).toEqual(["mid1", "mid2", "s1"]);
      expect(children.get("mid1")).toEqual(["inner1", "s2", "s3"]);
      expect(children.get("side")).toEqual(["s8", "s9"]);
      expect(children.has("l1")).toEqual(false);
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
      const selected = withMembers(["outer"], forestParentOf);
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
      const selected = [...withMembers(["side"], forestParentOf), "l1"];
      expect(canGroup(selected, forestNodes, forestParentOf)).toEqual(true);
    });

    it("should ignore keys without nodes", () => {
      expect(canGroup(["s6", "edge1"], forestNodes, forestParentOf)).toEqual(false);
    });
  });

  describe("createActions", () => {
    const flatNodes = [
      node("a"),
      node("b", 100, 20),
      node("c", -40, 60),
      node("d", 15, -25),
      node("e", 70, 80),
    ];

    it("should insert a group anchored at the members' min corner", () => {
      const result = createActions({
        selected: ["c", "a", "e", "b", "d"],
        nodes: flatNodes,
      });
      const inserted = result?.actions[0];
      expect(inserted).toMatchObject({
        type: "set_node",
        setNode: {
          node: {
            position: { x: -60, y: -65 },
            zIndex: -1,
            config: { variant: "group_box" },
          },
        },
      });
      assert(inserted?.type === "set_node");
      const { key } = inserted.setNode.node;
      expect(result?.actions.slice(1)).toEqual(
        ["c", "a", "e", "b", "d"].map((m) =>
          schematic.set({ path: ["nodes", m, "parent"], value: key }),
        ),
      );
      expect(result?.selection).toEqual([key, "c", "a", "e", "b", "d"]);
    });

    it("should group a member's outermost group, not the member", () => {
      const result = createActions({
        selected: ["s6", "l1", "s8"],
        nodes: forestNodes,
      });
      expect(result?.actions[0]).toMatchObject({
        setNode: { node: { position: { x: -20, y: -40 } } },
      });
      expect(
        result?.actions.slice(1).map((a) => (a.type === "set" ? a.set.path[1] : null)),
      ).toEqual(["outer", "l1", "side"]);
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
      const result = createActions({ selected: ["l1"], nodes: forestNodes });
      expect(result).toBeNull();
    });

    it("should return null when the selection holds one group and its members", () => {
      const result = createActions({
        selected: withMembers(["side"], forestParentOf),
        nodes: forestNodes,
      });
      expect(result).toBeNull();
    });
  });

  describe("withMembers", () => {
    it("should add a selected group's members recursively", () => {
      expect(withMembers(["outer"], forestParentOf)).toEqual([
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
      expect(withMembers(["inner1", "l1", "side"], forestParentOf)).toEqual([
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
      expect(withMembers(["s6", "inner1"], forestParentOf)).toEqual([
        "s6",
        "inner1",
        "s7",
      ]);
    });
  });

  describe("fanOutMoves", () => {
    it("should apply a moved group's delta to every symbol inside it", () => {
      const actions = fanOutMoves(doc(forestNodes), [
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
      const actions = fanOutMoves(doc(forestNodes), [
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
      const actions = fanOutMoves(doc(forestNodes), [
        schematic.setNodePosition({ key: "mid1", position: { x: 20, y: 10 } }),
        schematic.setNodePosition({ key: "s2", position: { x: 0, y: 0 } }),
      ]);
      expect(actions).toHaveLength(6);
      expect(actions).not.toContainEqual(
        schematic.setNodePosition({ key: "s2", position: { x: 12, y: 2 } }),
      );
    });

    it("should fan out even when the delta is zero", () => {
      const actions = fanOutMoves(doc(forestNodes), [
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
      expect(fanOutMoves(doc(forestNodes), input)).toEqual(input);
    });

    it("should return batches without position changes unchanged", () => {
      const input = [schematic.removeNode({ key: "s2" })];
      expect(fanOutMoves(doc(forestNodes), input)).toBe(input);
    });
  });

  describe("lockMembers", () => {
    it("should mark every grouped symbol as non-draggable", () => {
      const locked = lockMembers(forestNodes, forestParentOf, forest);
      locked.forEach((n, i) => {
        if (forestParentOf.has(n.key))
          expect(n).toEqual({ ...forestNodes[i], draggable: false });
        else expect(n).toBe(forestNodes[i]);
      });
    });

    it("should return the input unchanged when there are no groups", () => {
      expect(lockMembers(forestNodes, new Map(), {})).toBe(forestNodes);
    });

    it("should mark a locked group's box as non-draggable", () => {
      const locked = lockMembers(forestNodes, forestParentOf, withLocked("side"));
      const byKey = new Map(locked.map((n) => [n.key, n]));
      expect(byKey.get("side")?.draggable).toBe(false);
      expect(byKey.get("outer")?.draggable).toBeUndefined();
      expect(byKey.get("l1")?.draggable).toBeUndefined();
    });
  });

  describe("closure", () => {
    it("should expand a deep member to its outermost group and its contents", () => {
      expect(closure(["s6"], forestParentOf)).toEqual([
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

    it("should pass loose symbols through", () => {
      expect(closure(["l1", "l2"], forestParentOf)).toEqual(["l1", "l2"]);
    });

    it("should not duplicate a group clicked alongside its member", () => {
      expect(closure(["s8", "side"], forestParentOf)).toEqual(["side", "s8", "s9"]);
    });

    it("should expand each root of a multi-group selection", () => {
      expect(closure(["s6", "l1", "s8"], forestParentOf)).toEqual([
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
  });

  describe("drillIn", () => {
    it("should select a grouped member alone", () => {
      expect(drillIn("s6", forestParentOf, forest)).toEqual(["s6"]);
    });

    it("should select a nested group with its members", () => {
      expect(drillIn("mid1", forestParentOf, forest)).toEqual([
        "mid1",
        "inner1",
        "s6",
        "s7",
        "s2",
        "s3",
      ]);
    });

    it("should not drill into an outermost group", () => {
      expect(drillIn("outer", forestParentOf, forest)).toBeNull();
    });

    it("should not drill into a loose symbol", () => {
      expect(drillIn("l1", forestParentOf, forest)).toBeNull();
    });

    it("should not drill into an unknown key", () => {
      expect(drillIn("nope", forestParentOf, forest)).toBeNull();
    });

    it("should not drill under a locked root group", () => {
      const configs = withLocked("outer");
      expect(drillIn("s6", forestParentOf, configs)).toBeNull();
      expect(drillIn("mid1", forestParentOf, configs)).toBeNull();
    });

    it("should not drill under a locked inner group", () => {
      const configs = withLocked("mid1");
      expect(drillIn("s2", forestParentOf, configs)).toBeNull();
      expect(drillIn("s6", forestParentOf, configs)).toBeNull();
      expect(drillIn("s4", forestParentOf, configs)).toEqual(["s4"]);
    });

    it("should still drill in a sibling unlocked group", () => {
      expect(drillIn("s8", forestParentOf, withLocked("outer"))).toEqual(["s8"]);
    });
  });

  describe("shielded", () => {
    it("should shield members of a locked group but not the box itself", () => {
      const keys = ["side", "s8", "s9", "l1"];
      expect(shielded(keys, forestParentOf, withLocked("side"))).toEqual(
        new Set(["s8", "s9"]),
      );
    });

    it("should shield the whole subtree of a locked outer group", () => {
      const keys = withMembers(["outer"], forestParentOf);
      expect(shielded(keys, forestParentOf, withLocked("outer"))).toEqual(
        new Set(keys.filter((k) => k !== "outer")),
      );
    });

    it("should shield nothing when no group is locked", () => {
      expect(
        shielded(withMembers(["side"], forestParentOf), forestParentOf, forest),
      ).toEqual(new Set());
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
      const result = ungroupActions(withMembers(["side"], forestParentOf), forestNodes);
      expect(result?.actions).toEqual([schematic.removeNode({ key: "side" })]);
      expect(result?.freed).toEqual(["s8", "s9"]);
    });

    it("should target a selected member's immediate group", () => {
      const result = ungroupActions(["s6"], forestNodes);
      expect(result?.actions).toEqual([schematic.removeNode({ key: "inner1" })]);
      expect(result?.freed).toEqual(["s6", "s7"]);
    });

    it("should remove only the outermost group of a closed selection", () => {
      const result = ungroupActions(
        withMembers(["outer"], forestParentOf),
        forestNodes,
      );
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
      const result = ungroupActions(withMembers(["mid1"], forestParentOf), forestNodes);
      expect(result?.actions).toEqual([schematic.removeNode({ key: "mid1" })]);
      expect(result?.freed).toEqual(["inner1", "s2", "s3", "s6", "s7"]);
      const parentOf = buildParentOf(reduce(forestNodes, result?.actions ?? []));
      expect(parentOf.get("inner1")).toEqual("outer");
      expect(parentOf.get("s2")).toEqual("outer");
      expect(parentOf.get("s6")).toEqual("inner1");
    });

    it("should peel a chain one layer per ungroup", () => {
      let nodes = forestNodes;
      let selected = withMembers(["outer"], forestParentOf);
      const layers: string[][] = [];
      while (layers.length < 3) {
        const result = ungroupActions(selected, nodes);
        expect(result).not.toBeNull();
        if (result == null) break;
        layers.push(
          result.actions.flatMap((a) =>
            a.type === "remove_node" ? [a.removeNode.key] : [],
          ),
        );
        nodes = reduce(nodes, result.actions);
        selected = result.freed;
      }
      expect(layers).toEqual([["outer"], ["mid1", "mid2"], ["inner1"]]);
      expect(ungroupActions(selected, nodes)).toBeNull();
    });

    it("should return null when the selection touches no group", () => {
      expect(ungroupActions(["l1", "l2"], forestNodes)).toBeNull();
    });
  });

  describe("grouping cycle", () => {
    it("should release members on ungroup and reclaim them on regroup", () => {
      let nodes = [group("g"), node("a", 0, 0, "g"), node("b", 50, 50, "g")];
      const ungrouped = ungroupActions(["g", "a", "b"], nodes);
      nodes = reduce(nodes, ungrouped?.actions ?? []);
      expect(buildParentOf(nodes).has("a")).toEqual(false);
      const regrouped = createActions({ selected: ["a", "b"], nodes });
      const action = regrouped?.actions[0];
      assert(action?.type === "set_node");
      nodes = reduce(nodes, regrouped?.actions ?? []);
      expect(buildParentOf(nodes).get("a")).toEqual(action.setNode.node.key);
    });

    it("should keep an inner group's members after the outer ungroups", () => {
      const nodes = [
        group("outer"),
        group("inner", 0, 0, "outer"),
        node("c", 0, 0, "outer"),
        node("a", 0, 0, "inner"),
      ];
      const result = ungroupActions(
        withMembers(["outer"], buildParentOf(nodes)),
        nodes,
      );
      const parentOf = buildParentOf(reduce(nodes, result?.actions ?? []));
      expect(parentOf.get("a")).toEqual("inner");
      expect(parentOf.has("inner")).toEqual(false);
    });
  });
});
