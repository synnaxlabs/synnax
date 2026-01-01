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

import { device } from "@/device";
import { AuthError, NotFoundError } from "@/errors";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("device", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const rack = await client.racks.create({
        name: "test",
      });
      const randomDevice = await client.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "test",
        properties: {},
      });
      await expect(
        userClient.devices.retrieve({ key: randomDevice.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve devices with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [device.ontologyID("")],
        actions: ["retrieve"],
      });
      const rack = await client.racks.create({
        name: "test",
      });
      const randomDevice = await client.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "test",
        properties: {},
      });
      const retrieved = await userClient.devices.retrieve({
        key: randomDevice.key,
      });
      expect(retrieved.key).toBe(randomDevice.key);
      expect(retrieved.name).toBe(randomDevice.name);
    });

    it("should allow the caller to create devices with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [device.ontologyID("")],
        actions: ["create"],
      });
      const rack = await client.racks.create({
        name: "test",
      });
      await userClient.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "test",
        properties: {},
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [device.ontologyID("")],
        actions: [],
      });
      const rack = await client.racks.create({
        name: "test",
      });
      await expect(
        userClient.devices.create({
          key: id.create(),
          rack: rack.key,
          location: "Dev1",
          name: "test",
          make: "ni",
          model: "test",
          properties: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete devices with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [device.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const rack = await client.racks.create({
        name: "test",
      });
      const randomDevice = await client.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "test",
        properties: {},
      });
      await userClient.devices.delete(randomDevice.key);
      await expect(
        userClient.devices.retrieve({ key: randomDevice.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [device.ontologyID("")],
        actions: [],
      });
      const rack = await client.racks.create({
        name: "test",
      });
      const randomDevice = await client.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "test",
        properties: {},
      });
      await expect(userClient.devices.delete(randomDevice.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
