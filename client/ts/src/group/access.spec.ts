// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { group } from "@/group";
import { AccessDeniedError } from "@/index";
import { ontology } from "@/ontology";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("group", () => {
  describe("access control", () => {
    it("should allow the caller to create groups with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [group.resourceID("")],
        actions: ["create"],
      });
      await userClient.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [group.resourceID("")],
        actions: [],
      });
      await expect(
        userClient.groups.create({
          parent: ontology.ROOT_ID,
          name: `test-${id.create()}`,
        }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should allow the caller to delete groups with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [group.resourceID("")],
        actions: ["delete", "retrieve"],
      });
      const randomGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
      await userClient.groups.delete(randomGroup.key);
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [group.resourceID("")],
        actions: [],
      });
      const randomGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
      await expect(userClient.groups.delete(randomGroup.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should retrieve a page of children past a child the caller cannot read", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
      const parentID = group.resourceID(parent.key);
      await client.groups.create({ parent: parentID, name: "a" });
      await client.groups.create({ parent: parentID, name: "b" });
      const [first] = await client.groups.retrieve({ parent: parentID, limit: 1 });
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [group.resourceID(first.key)],
        actions: ["retrieve"],
      });
      const page = await userClient.groups.retrieve({ parent: parentID, limit: 1 });
      expect(page).toEqual([first]);
    });
  });
});
