// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { ranger } from "@/ranger";
import { createClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("range", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve ranges with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomRange = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
      await expect(userClient.ranges.retrieve(randomRange.key)).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve ranges with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [ranger.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomRange = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
      const retrieved = await userClient.ranges.retrieve(randomRange.key);
      expect(retrieved.key).toBe(randomRange.key);
      expect(retrieved.name).toBe(randomRange.name);
    });

    it("should allow the caller to create ranges with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [ranger.ontologyID("")],
        actions: ["create"],
      });
      await userClient.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
    });

    it("should prevent the caller to create ranges with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [ranger.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.ranges.create({
          name: "test",
          timeRange: new TimeRange(1n, 1000n),
          color: "#E774D0",
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete ranges with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [ranger.ontologyID("")],
        actions: ["delete"],
      });
      const randomRange = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
      await userClient.ranges.delete(randomRange.key);
      await expect(userClient.ranges.retrieve(randomRange.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should prevent the caller to delete ranges with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [ranger.ontologyID("")],
        actions: ["delete"],
      });
      const randomRange = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
      await expect(userClient.ranges.delete(randomRange.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
