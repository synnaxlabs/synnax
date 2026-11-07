// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { AccessDeniedError, NotFoundError, ValidationError } from "@/index";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("channel", () => {
  describe("access control", () => {
    it("should deny access when no matching policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const randomChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await expect(userClient.channels.retrieve(randomChannel.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should allow the caller to retrieve channels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.resourceID(0)],
        actions: ["retrieve"],
      });
      const randomChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const retrieved = await userClient.channels.retrieve(randomChannel.key);
      expect(retrieved.key).toBe(randomChannel.key);
      expect(retrieved.name).toBe(randomChannel.name);
      expect(retrieved.dataType.equals(randomChannel.dataType)).toBe(true);
    });

    it("should allow the caller to create channels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.resourceID(0)],
        actions: ["create"],
      });
      await userClient.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.resourceID(0)],
        actions: ["retrieve"],
      });
      await expect(
        userClient.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT32,
          virtual: true,
        }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should allow the caller to delete channels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.resourceID(0)],
        actions: ["delete"],
      });
      const randomChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await userClient.channels.delete(randomChannel.key);
      await expect(userClient.channels.retrieve(randomChannel.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.resourceID(0)],
        actions: ["retrieve"],
      });
      const randomChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await expect(userClient.channels.delete(randomChannel.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should show the key of a task the caller cannot retrieve", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      const rack = await client.racks.create({ name: "test" });
      const device = await client.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "127.0.0.1:502",
        name: "plc",
        make: "modbus",
        model: "modbus",
        properties: {},
      });
      const task = await rack.createTask({
        name: "read",
        type: "modbus_read",
        config: {
          device: device.key,
          channels: [{ type: "coil", key: "coil", channel: ch.key }],
        },
      });
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.TYPE_RESOURCE_ID],
        actions: ["retrieve", "delete"],
      });
      await expect(userClient.channels.delete(ch.key)).rejects.toThrow(
        new ValidationError(
          `cannot delete channel ${ch.name} because task ${task.key} uses it: ` +
            "validation error",
        ),
      );
      await client.tasks.delete(task.key);
      await client.channels.delete(ch.key);
    });
  });
});
