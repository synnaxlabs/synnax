// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { createClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";
import { workspace } from "@/workspace";

const client = createTestClient();

describe("workspace", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve workspaces with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomWorkspace = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(userClient.workspaces.retrieve(randomWorkspace.key)).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve workspaces with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [workspace.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomWorkspace = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const retrieved = await userClient.workspaces.retrieve(randomWorkspace.key);
      expect(retrieved.key).toBe(randomWorkspace.key);
      expect(retrieved.name).toBe(randomWorkspace.name);
    });

    it("should allow the caller to create workspaces with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [workspace.ontologyID("")],
        actions: ["create"],
      });
      await userClient.workspaces.create({
        name: "test",
        layout: {},
      });
    });

    it("should prevent the caller to create workspaces with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [workspace.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.workspaces.create({
          name: "test",
          layout: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete workspaces with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [workspace.ontologyID("")],
        actions: ["delete"],
      });
      const randomWorkspace = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.workspaces.delete(randomWorkspace.key);
      await expect(userClient.workspaces.retrieve(randomWorkspace.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should prevent the caller to delete workspaces with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [workspace.ontologyID("")],
        actions: ["delete"],
      });
      const randomWorkspace = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(userClient.workspaces.delete(randomWorkspace.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
