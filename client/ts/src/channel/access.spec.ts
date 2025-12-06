// Copyright 2025 Synnax Labs, Inc.
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
import { AuthError, NotFoundError } from "@/errors";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

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
      await expect(userClient.channels.retrieve(randomChannel.key)).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve channels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.ontologyID(0)],
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
        objects: [channel.ontologyID(0)],
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
        objects: [channel.ontologyID(0)],
        actions: ["retrieve"],
      });
      await expect(
        userClient.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT32,
          virtual: true,
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete channels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [channel.ontologyID(0)],
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
        objects: [channel.ontologyID(0)],
        actions: ["retrieve"],
      });
      const randomChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await expect(userClient.channels.delete(randomChannel.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
