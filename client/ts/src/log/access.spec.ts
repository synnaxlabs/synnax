// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AccessDeniedError, NotFoundError } from "@/index";
import { log } from "@/log";
import { project } from "@/project";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("log", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const proj = await client.projects.create({ name: "test" });
      const randomLog = await client.logs.create({ project: proj.key, name: "test" });
      await expect(userClient.logs.retrieve(randomLog.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should allow the caller to retrieve logs with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.resourceID("")],
        actions: ["retrieve"],
      });
      const proj = await client.projects.create({ name: "test" });
      const randomLog = await client.logs.create({ project: proj.key, name: "test" });
      const retrieved = await userClient.logs.retrieve(randomLog.key);
      expect(retrieved.key).toBe(randomLog.key);
      expect(retrieved.name).toBe(randomLog.name);
    });

    it("should allow the caller to create logs with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.resourceID(""), project.TYPE_RESOURCE_ID],
        actions: ["create", "update"],
      });
      const proj = await client.projects.create({ name: "test" });
      await userClient.logs.create({ project: proj.key, name: "test" });
    });

    it("should deny creating a log in a project the caller cannot update", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.resourceID("")],
        actions: ["create"],
      });
      const proj = await client.projects.create({ name: "test" });
      await expect(
        userClient.logs.create({ project: proj.key, name: "test" }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.resourceID("")],
        actions: [],
      });
      const proj = await client.projects.create({ name: "test" });
      await expect(
        userClient.logs.create({ project: proj.key, name: "test" }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should allow the caller to delete logs with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.resourceID("")],
        actions: ["delete", "retrieve"],
      });
      const proj = await client.projects.create({ name: "test" });
      const randomLog = await client.logs.create({ project: proj.key, name: "test" });
      await userClient.logs.delete(randomLog.key);
      await expect(userClient.logs.retrieve(randomLog.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.resourceID("")],
        actions: [],
      });
      const proj = await client.projects.create({ name: "test" });
      const randomLog = await client.logs.create({ project: proj.key, name: "test" });
      await expect(userClient.logs.delete(randomLog.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });
  });
});
