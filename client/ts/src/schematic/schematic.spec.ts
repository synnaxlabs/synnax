// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { schematic } from "@/schematic";
import { createTestClient } from "@/testutil/client";

const newWorkspaceSchematic = async (client: ReturnType<typeof createTestClient>) => {
  const ws = await client.workspaces.create({ name: "dispatch", layout: {} });
  const schem = await client.schematics.create(ws.key, {
    ...schematic.ZERO_NEW,
    name: "dispatch",
  });
  return { ws, schem };
};

const client = createTestClient();

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(ws.key, {
        ...schematic.ZERO_NEW,
        name: "Schematic",
      });
      expect(schem.name).toEqual("Schematic");
      expect(schem.key).not.toEqual(uuid.ZERO);
      const retrieved = await client.schematics.retrieve({ key: schem.key });
      expect(retrieved.key).toEqual(schem.key);
    });
  });

  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(ws.key, {
        ...schematic.ZERO_NEW,
        name: "Schematic",
      });
      await client.schematics.rename(schem.key, "Schematic2");
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.name).toEqual("Schematic2");
    });
  });

  describe("setData", () => {
    test("set data replaces body fields while preserving key and name", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(ws.key, {
        ...schematic.ZERO_NEW,
        name: "Schematic",
      });
      await client.schematics.setData(schem.key, {
        ...schematic.ZERO_NEW,
        authority: 5,
        nodes: [{ key: "n1", position: { x: 10, y: 20 }, zIndex: 0 }],
        props: { n1: { variant: "valve" } },
      });
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.name).toEqual("Schematic");
      expect(res.authority).toEqual(5);
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0].key).toEqual("n1");
      expect(res.props.n1.variant).toEqual("valve");
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(ws.key, {
        ...schematic.ZERO_NEW,
        name: "Schematic",
      });
      await client.schematics.delete(schem.key);
      await expect(client.schematics.retrieve({ key: schem.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("props case preservation", () => {
    test("preserves arbitrary key casing within prop values", async () => {
      const ws = await client.workspaces.create({ name: "CaseTest", layout: {} });
      const schem = await client.schematics.create(ws.key, {
        ...schematic.ZERO_NEW,
        props: {
          n1: {
            camelCaseKey: "value1",
            PascalCaseKey: "value2",
            snake_case_key: "value3",
            nested: {
              innerCamelCase: 123,
              InnerPascalCase: { deepKey: true },
            },
          },
        },
      });
      const retrieved = await client.schematics.retrieve({ key: schem.key });
      const props = retrieved.props.n1;
      expect(props.camelCaseKey).toEqual("value1");
      expect(props.PascalCaseKey).toEqual("value2");
      expect(props.snake_case_key).toEqual("value3");
      const nested = props.nested as Record<string, unknown>;
      expect(nested.innerCamelCase).toEqual(123);
      expect((nested.InnerPascalCase as Record<string, unknown>).deepKey).toEqual(true);
    });
  });

  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(ws.key, {
        ...schematic.ZERO_NEW,
        name: "Schematic",
      });
      const schem2 = await client.schematics.copy({
        key: schem.key,
        name: "Schematic2",
        snapshot: false,
      });
      expect(schem2.name).toEqual("Schematic2");
      expect(schem2.key).not.toEqual(uuid.ZERO);
    });

    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "Schematic",
          layout: { one: 1 },
        });
        const schem = await client.schematics.create(ws.key, {
          ...schematic.ZERO_NEW,
          name: "Schematic",
        });
        const schem2 = await client.schematics.copy({
          key: schem.key,
          name: "Schematic2",
          snapshot: true,
        });
        await expect(
          client.schematics.setData(schem2.key, {
            ...schematic.ZERO_NEW,
            authority: 2,
          }),
        ).rejects.toThrow(ValidationError);
      });
    });
  });

  describe("dispatch", () => {
    test("setNodePosition moves the matching node", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.setData(schem.key, {
        ...schematic.ZERO_NEW,
        nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
      });
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0].position).toEqual({ x: 100, y: 200 });
    });

    test("addNode appends a node and writes its props", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.addNode({
          node: { key: "n1", position: { x: 1, y: 2 } },
          props: { label: "Pump" },
        }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0]).toMatchObject({ key: "n1", position: { x: 1, y: 2 } });
      expect(res.props.n1.label).toBe("Pump");
    });

    test("removeNode removes the node and drops its props", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.setData(schem.key, {
        ...schematic.ZERO_NEW,
        nodes: [
          { key: "n1", position: { x: 0, y: 0 } },
          { key: "n2", position: { x: 1, y: 1 } },
        ],
        props: { n1: { label: "Pump" }, n2: { label: "Tank" } },
      });
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.removeNode({ key: "n1" }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0]).toMatchObject({ key: "n2", position: { x: 1, y: 1 } });
      expect(res.props).toEqual({ n2: { label: "Tank" } });
    });

    test("setEdge upserts an edge by key, replacing in place", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      const e = (
        key: string,
        srcNode: string,
        srcParam: string,
        tgtNode: string,
        tgtParam: string,
      ) => ({
        key,
        source: { node: srcNode, param: srcParam },
        target: { node: tgtNode, param: tgtParam },
      });
      await client.schematics.setData(schem.key, {
        ...schematic.ZERO_NEW,
        edges: [e("e1", "a", "o", "b", "i"), e("e2", "b", "o", "c", "i")],
      });
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setEdge({ edge: e("e2", "x", "y", "z", "w") }),
        schematic.setEdge({ edge: e("e3", "c", "o", "d", "i") }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.edges).toEqual([
        e("e1", "a", "o", "b", "i"),
        e("e2", "x", "y", "z", "w"),
        e("e3", "c", "o", "d", "i"),
      ]);
    });

    test("removeEdge removes the matching edge", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.setData(schem.key, {
        ...schematic.ZERO_NEW,
        edges: [
          {
            key: "e1",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
          },
        ],
      });
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.removeEdge({ key: "e1" }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.edges).toEqual([]);
    });

    test("setProps upserts props under the given key", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setProps({ key: "n1", props: { label: "Original" } }),
        schematic.setProps({ key: "n1", props: { label: "Replaced" } }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.props.n1.label).toBe("Replaced");
    });

    test("setAuthority replaces the authority value", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setAuthority({ value: 200 }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.authority).toBe(200);
    });

    test("setLegend replaces the legend wholesale", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      const legend = {
        visible: true,
        position: { x: 10, y: 20 },
        colors: { on: color.construct("#ff0000") },
      };
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setLegend({ legend }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.legend).toEqual(legend);
    });

    test("applies a multi-action sequence atomically", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.addNode({ node: { key: "pump", position: { x: 0, y: 0 } } }),
        schematic.addNode({ node: { key: "valve", position: { x: 100, y: 0 } } }),
        schematic.setEdge({
          edge: {
            key: "e1",
            source: { node: "pump", param: "out" },
            target: { node: "valve", param: "in" },
          },
        }),
        schematic.setProps({ key: "pump", props: { label: "Main Pump" } }),
        schematic.setAuthority({ value: 200 }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(2);
      expect(res.edges).toHaveLength(1);
      expect(res.authority).toBe(200);
      expect(res.props.pump.label).toBe("Main Pump");
    });

    test("converges to the final position after a 30-action drag storm", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.setData(schem.key, {
        ...schematic.ZERO_NEW,
        nodes: [{ key: "pump", position: { x: 0, y: 0 } }],
      });
      const actions = Array.from({ length: 30 }, (_, i) =>
        schematic.setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }),
      );
      await client.schematics.dispatch(schem.key, "sess-1", actions);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    test("rejects dispatch on a snapshot with ValidationError", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      const snap = await client.schematics.copy({
        key: schem.key,
        name: "snap",
        snapshot: true,
      });
      await expect(
        client.schematics.dispatch(snap.key, "sess-1", [
          schematic.setAuthority({ value: 9 }),
        ]),
      ).rejects.toThrow(ValidationError);
    });

    test("rejects dispatch with NotFoundError when the target schematic is missing", async () => {
      await expect(
        client.schematics.dispatch(uuid.create(), "sess-1", [
          schematic.setAuthority({ value: 9 }),
        ]),
      ).rejects.toThrow(NotFoundError);
    });

    test("preserves arbitrary key casing within prop values through dispatch", async () => {
      const { schem } = await newWorkspaceSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setProps({
          key: "n1",
          props: {
            camelCaseKey: "v1",
            PascalCaseKey: "v2",
            snake_case_key: "v3",
            nested: { innerCamelCase: 1, InnerPascalCase: { deepKey: true } },
          },
        }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      const props = res.props.n1;
      expect(props.camelCaseKey).toBe("v1");
      expect(props.PascalCaseKey).toBe("v2");
      expect(props.snake_case_key).toBe("v3");
      const nested = props.nested as Record<string, unknown>;
      expect(nested.innerCamelCase).toBe(1);
      expect((nested.InnerPascalCase as Record<string, unknown>).deepKey).toBe(true);
    });
  });
});
