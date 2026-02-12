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
import { log } from "@/log";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("log", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(userClient.logs.retrieve({ key: randomLog.key })).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve logs with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.ontologyID("")],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      const retrieved = await userClient.logs.retrieve({
        key: randomLog.key,
      });
      expect(retrieved.key).toBe(randomLog.key);
      expect(retrieved.name).toBe(randomLog.name);
    });

    it("should allow the caller to create logs with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.logs.create(ws.key, {
        name: "test",
        data: {},
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.logs.create(ws.key, {
          name: "test",
          data: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete logs with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      await userClient.logs.delete(randomLog.key);
      await expect(userClient.logs.retrieve({ key: randomLog.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [log.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLog = await client.logs.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(userClient.logs.delete(randomLog.key)).rejects.toThrow(AuthError);
    });
  });
});
