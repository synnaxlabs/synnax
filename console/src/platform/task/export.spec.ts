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
import { Session } from "@/session";
import { createTestStore, type TestStore, uniqueName } from "@/testutil";

const client: Synnax = createTestClient();

const createTask = async (config: Record<string, unknown> = { sampleRate: 100 }) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({ name: "Read Task", type: "testReadType", config });
};

const placeTaskLayout = (store: TestStore, args?: Record<string, unknown>): string => {
  const key = `task-layout-${id.create()}`;
  store.dispatch(
    Session.Layout.place({
      windowKey: "main",
      key,
      type: "task",
      name: "Configure",
      location: "mosaic",
    }),
  );
  if (args != null) store.dispatch(Session.Layout.setArgs({ key, args }));
  return key;
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

    it("should resolve the task key from a task layout's args", async () => {
      const store = await createTestStore();
      const t = await createTask({ sampleRate: 25 });
      const layoutKey = placeTaskLayout(store, { taskKey: t.key });
      const file = await Task.extract(layoutKey, { client, store });
      expect(file.name).toBe("Read Task");
      expect(JSON.parse(file.data)).toEqual({ sampleRate: 25, type: "testReadType" });
    });

    it("should throw when the key is neither a task nor a task layout", async () => {
      const store = await createTestStore();
      await expect(
        Task.extract(`missing-${id.create()}`, { client, store }),
      ).rejects.toThrow("neither the key of a task nor the key of a task layout");
    });

    it("should throw when the layout has no configured task key", async () => {
      const store = await createTestStore();
      const layoutKey = placeTaskLayout(store);
      await expect(Task.extract(layoutKey, { client, store })).rejects.toThrow(
        "You should configure the task before exporting it",
      );
    });
  });
});
