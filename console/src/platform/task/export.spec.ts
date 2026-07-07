// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DisconnectedError, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { createTestStore, uniqueName } from "@/testutil";

const client: Synnax = createTestClient();

const createTask = async (config: Record<string, unknown> = { sampleRate: 100 }) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({ name: "Read Task", type: "testReadType", config });
};

describe("task export", () => {
  describe("extract", () => {
    it("should throw a DisconnectedError when the client is null", async () => {
      const store = await createTestStore();
      const err = await Task.extract("123", { client: null, store }).catch(
        (e: unknown) => e,
      );
      expect(DisconnectedError.matches(err)).toBe(true);
    });

    it("should extract a task directly from a numeric task key", async () => {
      const store = await createTestStore();
      const t = await createTask();
      const file = await Task.extract(t.key, { client, store });
      expect(file.name).toBe("Read Task");
      expect(JSON.parse(file.data)).toEqual({ sampleRate: 100, type: "testReadType" });
    });

    it("should throw when the key is not a task key", async () => {
      const store = await createTestStore();
      await expect(
        Task.extract(`missing-${id.create()}`, { client, store }),
      ).rejects.toThrow("You should configure the task before exporting it");
    });
  });
});
