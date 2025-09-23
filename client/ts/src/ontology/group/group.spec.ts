// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { ontology } from "@/ontology";
import { group } from "@/ontology/group";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Group", () => {
  describe("create", () => {
    it("should correctly create a group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.ontology.groups.create({ parent: ontology.ROOT_ID, name });
      expect(g.name).toEqual(name);
    });
  });
  describe("rename", () => {
    it("should correctly rename a group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.ontology.groups.create({ parent: ontology.ROOT_ID, name });
      const newName = `group-${Math.random()}`;
      await client.ontology.groups.rename(g.key, newName);
      const g2 = await client.ontology.retrieve(group.ontologyID(g.key));
      expect(g2.name).toEqual(newName);
    });
  });
  describe("delete", () => {
    it("should correctly delete the group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.ontology.groups.create({ parent: ontology.ROOT_ID, name });
      await client.ontology.groups.delete(g.key);
      await expect(
        async () => await client.ontology.retrieve(group.ontologyID(g.key)),
      ).rejects.toThrowError(NotFoundError);
    });
  });
});
