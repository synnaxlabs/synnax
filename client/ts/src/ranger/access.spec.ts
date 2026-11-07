// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import { AccessDeniedError, NotFoundError } from "@/index";
import { label } from "@/label";
import { ranger } from "@/ranger";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("range", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const randomRange = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
      await expect(userClient.ranges.retrieve(randomRange.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should allow the caller to retrieve ranges with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.resourceID("")],
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
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.resourceID("")],
        actions: ["create"],
      });
      await userClient.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.resourceID("")],
        actions: [],
      });
      await expect(
        userClient.ranges.create({
          name: "test",
          timeRange: new TimeRange(1n, 1000n),
          color: "#E774D0",
        }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should allow the caller to delete ranges with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.resourceID("")],
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

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.resourceID("")],
        actions: [],
      });
      const randomRange = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(1n, 1000n),
        color: "#E774D0",
      });
      await expect(userClient.ranges.delete(randomRange.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should give the label keys of a range to a caller that cannot read the labels", async () => {
      const lbl = await client.labels.create({ name: "test", color: "#FF0000" });
      const onError = vi.fn();
      const userClient = await createTestClientWithPolicy(
        client,
        [
          {
            name: "ranges",
            objects: [
              channel.TYPE_RESOURCE_ID,
              framer.TYPE_RESOURCE_ID,
              ranger.TYPE_RESOURCE_ID,
            ],
            actions: ["retrieve", "update"],
          },
          { name: "labels", objects: [label.TYPE_RESOURCE_ID], actions: ["update"] },
        ],
        onError,
      );
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange({ start: 1, end: 2 }),
      });
      await userClient.labels.label(ranger.resourceID(rng.key), [lbl.key]);
      const retrieved = await userClient.ranges.retrieve(rng.key);
      expect(retrieved.labels).toEqual([lbl.key]);
      await expect(userClient.labels.retrieve({ key: lbl.key })).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
      expect(onError).not.toHaveBeenCalled();
    });
  });
});
