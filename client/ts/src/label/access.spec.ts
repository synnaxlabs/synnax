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
import { label } from "@/label";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("label", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve labels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomLabel = await client.labels.create({
        name: "test",
        color: "#E774D0",
      });
      await expect(
        userClient.labels.retrieve({ key: randomLabel.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve labels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [label.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomLabel = await client.labels.create({
        name: "test",
        color: "#E774D0",
      });
      const retrieved = await userClient.labels.retrieve({ key: randomLabel.key });
      expect(retrieved.key).toBe(randomLabel.key);
      expect(retrieved.name).toBe(randomLabel.name);
      expect(retrieved.color).toBe(randomLabel.color);
    });

    it("should allow the caller to create labels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [label.ontologyID("")],
        actions: ["create"],
      });
      await userClient.labels.create({
        name: "test",
        color: "#E774D0",
      });
    });

    it("should prevent the caller to create labels with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [label.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.labels.create({
          name: "test",
          color: "#E774D0",
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete labels with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [label.ontologyID("")],
        actions: ["delete"],
      });
      const randomLabel = await client.labels.create({
        name: "test",
        color: "#E774D0",
      });
      await userClient.labels.delete(randomLabel.key);
      await expect(
        userClient.labels.retrieve({ key: randomLabel.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete labels with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [label.ontologyID("")],
        actions: ["delete"],
      });
      const randomLabel = await client.labels.create({
        name: "test",
        color: "#E774D0",
      });
      await expect(userClient.labels.delete(randomLabel.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
