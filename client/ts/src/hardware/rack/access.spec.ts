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
import { createClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("rack", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve racks with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomRack = await client.hardware.racks.create({
        name: "test",
      });
      await expect(
        userClient.hardware.racks.retrieve({ key: randomRack.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve racks with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [rack.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomRack = await client.hardware.racks.create({
        name: "test",
      });
      const retrieved = await userClient.hardware.racks.retrieve({
        key: randomRack.key,
      });
      expect(retrieved.key).toBe(randomRack.key);
      expect(retrieved.name).toBe(randomRack.name);
    });

    it("should allow the caller to create racks with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [rack.ontologyID("")],
        actions: ["create"],
      });
      await userClient.hardware.racks.create({
        name: "test",
      });
    });

    it("should prevent the caller to create racks with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [rack.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.hardware.racks.create({
          name: "test",
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete racks with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [rack.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const randomRack = await client.hardware.racks.create({
        name: "test",
      });
      await userClient.hardware.racks.delete(randomRack.key);
      await expect(
        userClient.hardware.racks.retrieve({ key: randomRack.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete racks with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [rack.ontologyID("")],
        actions: ["delete"],
      });
      const randomRack = await client.hardware.racks.create({
        name: "test",
      });
      await expect(userClient.hardware.racks.delete(randomRack.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
