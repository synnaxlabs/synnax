// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { addNode } from "@/schematic/reducer";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        version: 1,
        nodes: [],
        edges: [],
        props: {},
      });
      expect(schematic.name).toEqual("Schematic");
      expect(schematic.key).not.toEqual(uuid.ZERO);
      expect(schematic.version).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        version: 1,
        nodes: [],
        edges: [],
        props: {},
      });
      await client.workspaces.schematics.rename(schematic.key, "Schematic2");
      const res = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });
      expect(res.name).toEqual("Schematic2");
    });
  });
  describe("update", () => {
    test("add node", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        version: 1,
        nodes: [],
        edges: [],
        props: {},
      });
      await client.workspaces.schematics.update(
        schematic.key,
        addNode({
          key: "node1",
          node: { key: "node1", position: { x: 0, y: 0 }, type: "test" },
        }),
      );
      const res = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0].key).toEqual("node1");
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        version: 1,
        nodes: [],
        edges: [],
        props: {},
      });
      await client.workspaces.schematics.delete(schematic.key);
      await expect(
        client.workspaces.schematics.retrieve({ key: schematic.key }),
      ).rejects.toThrow(NotFoundError);
    });
  });
  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        version: 1,
        nodes: [],
        edges: [],
        props: {},
      });
      const schematic2 = await client.workspaces.schematics.copy({
        key: schematic.key,
        name: "Schematic2",
        snapshot: false,
      });
      expect(schematic2.name).toEqual("Schematic2");
      expect(schematic2.key).not.toEqual(uuid.ZERO);
      expect(schematic2.version).toEqual(1);
    });
    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "Schematic",
          layout: { one: 1 },
        });
        const schematic = await client.workspaces.schematics.create(ws.key, {
          name: "Schematic",
          version: 1,
          nodes: [],
          edges: [],
          props: {},
        });
        const schematic2 = await client.workspaces.schematics.copy({
          key: schematic.key,
          name: "Schematic2",
          snapshot: true,
        });
        await expect(
          client.workspaces.schematics.update(
            schematic2.key,
            addNode({
              key: "node1",
              node: { key: "node1", position: { x: 0, y: 0 }, type: "test" },
            }),
          ),
        ).rejects.toThrow(ValidationError);
      });
    });
  });
});
