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
import { log } from "@/workspace/log";

const client = createTestClient();

describe("log", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve logs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.workspaces.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.workspaces.logs.retrieve({ key: randomLog.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve logs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [log.ontologyID("")],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.workspaces.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      const retrieved = await userClient.workspaces.logs.retrieve({
        key: randomLog.key,
      });
      expect(retrieved.key).toBe(randomLog.key);
      expect(retrieved.name).toBe(randomLog.name);
    });

    it("should allow the caller to create logs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [log.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.workspaces.logs.create(ws.key, {
        name: "test",
        data: {},
      });
    });

    it("should prevent the caller to create logs with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [log.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.workspaces.logs.create(ws.key, {
          name: "test",
          data: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete logs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [log.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.workspaces.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      await userClient.workspaces.logs.delete(randomLog.key);
      await expect(
        userClient.workspaces.logs.retrieve({ key: randomLog.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete logs with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [log.ontologyID("")],
        actions: ["delete"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.workspaces.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(userClient.workspaces.logs.delete(randomLog.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
