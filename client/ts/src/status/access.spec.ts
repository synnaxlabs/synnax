// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { status } from "@/status";
import { createClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("status", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve statuses with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomStatus = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      await expect(
        userClient.statuses.retrieve({ key: randomStatus.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve statuses with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [status.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomStatus = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      const retrieved = await userClient.statuses.retrieve({
        key: randomStatus.key,
      });
      expect(retrieved.key).toBe(randomStatus.key);
      expect(retrieved.name).toBe(randomStatus.name);
    });

    it("should allow the caller to set statuses with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [status.ontologyID("")],
        actions: ["create"],
      });
      await userClient.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
    });

    it("should prevent the caller to set statuses with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [status.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.statuses.set({
          name: "test",
          key: uuid.create(),
          variant: "info",
          message: "test",
          time: TimeStamp.now(),
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete statuses with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [status.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const randomStatus = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      await userClient.statuses.delete(randomStatus.key);
      await expect(
        userClient.statuses.retrieve({ key: randomStatus.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete statuses with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [status.ontologyID("")],
        actions: ["delete"],
      });
      const randomStatus = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      await expect(userClient.statuses.delete(randomStatus.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
