// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { project } from "@/project";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("project", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const randomProject = await client.projects.create({
        name: "test",
        layout: {},
      });
      await expect(userClient.projects.retrieve({ key: randomProject.key })).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve projects with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [project.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomProject = await client.projects.create({
        name: "test",
        layout: {},
      });
      const retrieved = await userClient.projects.retrieve({ key: randomProject.key });
      expect(retrieved.key).toBe(randomProject.key);
      expect(retrieved.name).toBe(randomProject.name);
    });

    it("should allow the caller to create projects with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [project.ontologyID("")],
        actions: ["create"],
      });
      await userClient.projects.create({
        name: "test",
        layout: {},
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [project.ontologyID("")],
        actions: [],
      });
      await expect(
        userClient.projects.create({
          name: "test",
          layout: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete projects with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [project.ontologyID("")],
        actions: ["delete"],
      });
      const randomProject = await client.projects.create({
        name: "test",
        layout: {},
      });
      await userClient.projects.delete(randomProject.key);
      await expect(userClient.projects.retrieve({ key: randomProject.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [project.ontologyID("")],
        actions: [],
      });
      const randomProject = await client.projects.create({
        name: "test",
        layout: {},
      });
      await expect(userClient.projects.delete(randomProject.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
