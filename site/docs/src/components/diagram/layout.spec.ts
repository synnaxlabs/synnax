// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { layout, type Node } from "@/components/diagram/layout";

const NODES: Node[] = [
  { key: "task", label: "Read task", column: 0 },
  { key: "script", label: "Script", column: 0 },
  { key: "button", label: "Schematic button", column: 0 },
  { key: "channel", label: "Channel", column: 1 },
];

const EDGES = NODES.slice(0, 3).map(({ key }) => ({ from: key, to: "channel" }));

const byKey = (key: string) => {
  const node = layout(NODES, EDGES).nodes.find((n) => n.key === key);
  if (node == null) throw new Error(`no node ${key}`);
  return node;
};

describe("layout", () => {
  it("should stack the nodes of a column in the given order", () => {
    expect(byKey("task").y).toBeLessThan(byKey("script").y);
    expect(byKey("script").y).toBeLessThan(byKey("button").y);
  });

  it("should give every node in a column the width of the widest one", () => {
    expect(byKey("task").width).toEqual(byKey("button").width);
  });

  it("should center a short column against a tall one", () => {
    const channel = byKey("channel");
    const script = byKey("script");
    expect(channel.y + channel.height / 2).toEqual(script.y + script.height / 2);
  });

  it("should place columns from left to right", () => {
    const task = byKey("task");
    expect(byKey("channel").x).toBeGreaterThan(task.x + task.width);
  });

  it("should make a node with a caption taller", () => {
    const { nodes } = layout(
      [
        { key: "a", label: "A", column: 0 },
        { key: "b", label: "B", caption: "second line", column: 0 },
      ],
      [],
    );
    expect(nodes[1].height).toBeGreaterThan(nodes[0].height);
  });

  it("should draw a straight line between nodes at the same height", () => {
    const { edges } = layout(NODES, EDGES);
    expect(edges.find(({ from }) => from === "script")?.path).not.toContain("Q");
  });

  it("should bend a line between nodes at different heights", () => {
    const { edges } = layout(NODES, EDGES);
    expect(edges.find(({ from }) => from === "task")?.path).toContain("Q");
  });

  it("should record which nodes have lines leaving and entering them", () => {
    const { sources, targets } = layout(NODES, EDGES);
    expect([...sources].sort()).toEqual(["button", "script", "task"]);
    expect([...targets]).toEqual(["channel"]);
  });

  it("should fit the size to the content", () => {
    const { width, height, nodes } = layout(NODES, EDGES);
    for (const node of nodes) {
      expect(node.x + node.width).toBeLessThan(width);
      expect(node.y + node.height).toBeLessThan(height);
    }
  });

  describe("down", () => {
    const placed = (key: string) => {
      const node = layout(NODES, EDGES, "down").nodes.find((n) => n.key === key);
      if (node == null) throw new Error(`no node ${key}`);
      return node;
    };

    it("should place the nodes of a column side by side", () => {
      const task = placed("task");
      const script = placed("script");
      expect(script.x).toBeGreaterThan(task.x + task.width);
      expect(script.y + script.height / 2).toEqual(task.y + task.height / 2);
    });

    it("should place columns from top to bottom", () => {
      const task = placed("task");
      expect(placed("channel").y).toBeGreaterThan(task.y + task.height);
    });

    it("should center a narrow rank under a wide one", () => {
      const channel = placed("channel");
      const { width } = layout(NODES, EDGES, "down");
      expect(channel.x + channel.width / 2).toEqual(width / 2);
    });

    it("should break the text of a rank too wide for a phone onto two lines", () => {
      const wide = ["Arc automation", "Python script", "Schematic button"].map(
        (label, i) => ({ key: `n${i}`, label, column: 0 }),
      );
      const { nodes } = layout(wide, [], "down");
      expect(nodes.map(({ label }) => label)).toEqual([
        "Arc\nautomation",
        "Python\nscript",
        "Schematic\nbutton",
      ]);
      expect(nodes[0].height).toBeGreaterThan(layout(wide, []).nodes[0].height);
    });

    it("should break a list after a comma", () => {
      const wide = ["Plugged-in devices", "Network devices"].map((label, i) => ({
        key: `n${i}`,
        label,
        caption: i === 0 ? "USB, PCIe" : "PLCs, OPC UA, HTTP",
        column: 0,
      }));
      const { nodes } = layout(wide, [], "down");
      expect(nodes[1].caption).toEqual("PLCs,\nOPC UA, HTTP");
    });

    it("should keep the text of a rank that fits on one line", () => {
      const { nodes } = layout(NODES.slice(0, 2), [], "down");
      expect(nodes[0].label).toEqual("Read task");
    });

    it("should draw a straight line between nodes above one another", () => {
      const { edges } = layout(
        [
          { key: "a", label: "A", column: 0 },
          { key: "b", label: "B", column: 1 },
        ],
        [{ from: "a", to: "b" }],
        "down",
      );
      expect(edges[0].path).not.toContain("Q");
    });

    it("should bend a line between nodes that are not above one another", () => {
      const { edges } = layout(NODES, EDGES, "down");
      expect(edges.find(({ from }) => from === "task")?.path).toContain("Q");
    });
  });

  it("should throw when an edge refers to an unknown node", () => {
    expect(() => layout(NODES, [{ from: "task", to: "nope" }])).toThrow(
      'diagram edge refers to unknown node "nope"',
    );
  });
});
