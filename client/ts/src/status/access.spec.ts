// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import { AccessDeniedError, NotFoundError } from "@/index";
import { status } from "@/status";
import {
  createTestClient,
  createTestClientWithPolicy,
  recordResponses,
  waitForStreamLive,
} from "@/testutil";

const client = createTestClient();

describe("status", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const randomStatus = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      await expect(userClient.statuses.retrieve(randomStatus.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should allow the caller to retrieve statuses with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
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
      const retrieved = await userClient.statuses.retrieve(randomStatus.key);
      expect(retrieved.key).toBe(randomStatus.key);
      expect(retrieved.name).toBe(randomStatus.name);
    });

    it("should allow the caller to set statuses with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
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

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [status.ontologyID("")],
        actions: [],
      });
      await expect(
        userClient.statuses.set({
          name: "test",
          key: uuid.create(),
          variant: "info",
          message: "test",
          time: TimeStamp.now(),
        }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should allow the caller to delete statuses with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
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
      await expect(userClient.statuses.retrieve(randomStatus.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [status.ontologyID("")],
        actions: [],
      });
      const randomStatus = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      await expect(userClient.statuses.delete(randomStatus.key)).rejects.toSatisfy(
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
            status.ontologyID(""),
          ],
          actions: ["retrieve"],
        },
        onError,
      );
      await waitForStreamLive(userClient.connection);
      const fetches = recordResponses(userClient, "/label/retrieve");
      const st = await client.statuses.set({
        name: "test",
        key: uuid.create(),
        variant: "info",
        message: "test",
        time: TimeStamp.now(),
      });
      onTestFinished(userClient.statuses.onChange(st.key, vi.fn()));
      await client.labels.label(status.ontologyID(st.key), [lbl.key]);
      await expect.poll(() => fetches.length).toBeGreaterThan(0);
      for (const fetch of fetches)
        await expect(fetch).rejects.toSatisfy(AccessDeniedError.matches);
      // Lets the rejection reach the error handler.
      await new Promise((resolve) => setTimeout(resolve, 0));
      expect(onError).not.toHaveBeenCalled();
    });
  });
});
