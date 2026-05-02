// Copyright 2026 Synnax Labs, Inc.
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
import { schematic } from "@/schematic";
import { createTestClient } from "@/testutil/client";

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
      expect((res.props.n1 as Record<string, unknown>).variant).toEqual("valve");
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
      const props = retrieved.props.n1 as Record<string, unknown>;
      expect(props.camelCaseKey).toEqual("value1");
      expect(props.PascalCaseKey).toEqual("value2");
      expect(props.snake_case_key).toEqual("value3");
      expect((props.nested as Record<string, unknown>).innerCamelCase).toEqual(123);
      expect(
        (
          (props.nested as Record<string, unknown>).InnerPascalCase as Record<
            string,
            unknown
          >
        ).deepKey,
      ).toEqual(true);
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
});
