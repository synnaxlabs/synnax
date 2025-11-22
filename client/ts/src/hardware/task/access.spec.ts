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
import { rack } from "@/hardware/rack";
import { task } from "@/hardware/task";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("task", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve tasks with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const randomTask = await rack.createTask({
        name: "test",
        type: "ni",
        config: {},
      });
      await expect(
        userClient.hardware.tasks.retrieve({ key: randomTask.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve tasks with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [task.ontologyID("")],
        actions: ["retrieve"],
      });
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const randomTask = await rack.createTask({
        name: "test",
        type: "ni",
        config: {},
      });
      const retrieved = await userClient.hardware.tasks.retrieve({
        key: randomTask.key,
      });
      expect(retrieved.key).toBe(randomTask.key);
      expect(retrieved.name).toBe(randomTask.name);
    });

    it("should allow the caller to create tasks with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [task.ontologyID(""), rack.ontologyID(0)],
        actions: ["create", "retrieve"],
      });
      const rck = await client.hardware.racks.create({
        name: "test",
      });
      const userRack = await userClient.hardware.racks.retrieve({ key: rck.key });
      await userRack.createTask({
        name: "test",
        type: "ni",
        config: {},
      });
    });

    it("should prevent the caller to create tasks with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [task.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.hardware.tasks.create({
          name: "test",
          type: "ni",
          config: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete tasks with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [task.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const randomTask = await rack.createTask({
        name: "test",
        type: "ni",
        config: {},
      });
      await userClient.hardware.tasks.delete(randomTask.key);
      await expect(
        userClient.hardware.tasks.retrieve({ key: randomTask.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete tasks with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [task.ontologyID("")],
        actions: ["delete"],
      });
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const randomTask = await rack.createTask({
        name: "test",
        type: "ni",
        config: {},
      });
      await expect(userClient.hardware.tasks.delete(randomTask.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
