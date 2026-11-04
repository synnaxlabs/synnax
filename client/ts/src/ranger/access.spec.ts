// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import { AccessDeniedError, NotFoundError } from "@/index";
import { ranger } from "@/ranger";
import {
  createTestClient,
  createTestClientWithPolicy,
  recordResponses,
  waitForStreamLive,
} from "@/testutil";

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
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.ontologyID("")],
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
        objects: [ranger.ontologyID("")],
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

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ranger.ontologyID("")],
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

    it("should report no error for a label the caller cannot read", async () => {
      // Created before the caller connects, so its stream never carries the label.
      const lbl = await client.labels.create({ name: "test", color: "#FF0000" });
      const onError = vi.fn();
      const userClient = await createTestClientWithPolicy(
        client,
        {
          name: "test",
          objects: [
            channel.TYPE_ONTOLOGY_ID,
            framer.TYPE_ONTOLOGY_ID,
            ranger.ontologyID(""),
          ],
          actions: ["retrieve"],
        },
        onError,
      );
      await waitForStreamLive(userClient.connection);
      const fetches = recordResponses(userClient, "/label/retrieve");
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange({ start: 1, end: 2 }),
      });
      onTestFinished(userClient.ranges.onChange(rng.key, vi.fn()));
      await client.labels.label(ranger.ontologyID(rng.key), [lbl.key]);
      await expect.poll(() => fetches.length).toBeGreaterThan(0);
      for (const fetch of fetches)
        await expect(fetch).rejects.toSatisfy(AccessDeniedError.matches);
      // Lets the rejection reach the error handler.
      await new Promise((resolve) => setTimeout(resolve, 0));
      expect(onError).not.toHaveBeenCalled();
    });
  });
});
