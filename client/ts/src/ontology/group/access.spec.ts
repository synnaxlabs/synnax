// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { AuthError } from "@/errors";
import { ontology } from "@/ontology";
import { group } from "@/ontology/group";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("group", () => {
  describe("access control", () => {
    it("should allow the caller to create groups with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [group.ontologyID("")],
        actions: ["create"],
      });
      await userClient.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
    });

    it("should prevent the caller to create groups with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [group.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.ontology.groups.create({
          parent: ontology.ROOT_ID,
          name: `test-${id.create()}`,
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete groups with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [group.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const randomGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
      await userClient.ontology.groups.delete(randomGroup.key);
    });

    it("should prevent the caller to delete groups with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [group.ontologyID("")],
        actions: ["delete"],
      });
      const randomGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-${id.create()}`,
      });
      await expect(userClient.ontology.groups.delete(randomGroup.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
