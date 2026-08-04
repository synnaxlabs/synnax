// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderHook } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { awaitTaskResourceTab, createSelectedPanel } from "@/platform/task/testutil";
import { createConsoleWrapper, createTestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const TYPE = "test_create";

const schemas = {
  type: z.literal(TYPE),
  config: z.object({ device: z.string().default("") }),
  statusData: z.unknown(),
};

const getInitialValues: Task.GetInitialValues<typeof schemas> = ({ deviceKey }) => ({
  name: "New Create Task",
  type: TYPE,
  config: { device: deviceKey ?? "" },
});

const useCreateTest = Task.createUseCreate({ getInitialValues });

const setup = async () => {
  const store = await createTestStore();
  const { wrapper } = await createConsoleWrapper({ client, store });
  const created = await createSelectedPanel(store, client);
  const { result } = renderHook(() => useCreateTest(), { wrapper });
  return { created, result };
};

describe("createUseCreate", () => {
  it("should create a draft row on rack zero and open its resource tab", async () => {
    const { created, result } = await setup();
    act(() => result.current({ deviceKey: "dev-1" }));
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key, schemas });
    expect(tsk.rack).toBe(0);
    expect(tsk.type).toBe(TYPE);
    expect(tsk.name).toBe("New Create Task");
    expect(tsk.config.device).toBe("dev-1");
  });

  it("should honor an explicit rack", async () => {
    const rack = await client.racks.create({ name: uniqueName("rack") });
    const { created, result } = await setup();
    act(() => result.current({ rackKey: rack.key }));
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key, schemas });
    expect(tsk.rack).toBe(rack.key);
  });

  it("should pass an explicit config through to getInitialValues", async () => {
    const useCreateFromConfig = Task.createUseCreate<typeof schemas>({
      getInitialValues: ({ config }) => ({
        name: "Config Create Task",
        type: TYPE,
        config: config as z.infer<(typeof schemas)["config"]>,
      }),
    });
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    const created = await createSelectedPanel(store, client);
    const { result } = renderHook(() => useCreateFromConfig(), { wrapper });
    act(() => result.current({ config: { device: "dev-2" } }));
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key, schemas });
    expect(tsk.config.device).toBe("dev-2");
  });

  it("should bind each minted hook to its own task type", async () => {
    const otherType = "test_create_other";
    const useCreateOther = Task.createUseCreate({
      getInitialValues: () => ({
        name: "Other Create Task",
        type: otherType,
        config: {},
      }),
    });
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    const created = await createSelectedPanel(store, client);
    const { result } = renderHook(() => useCreateOther(), { wrapper });
    act(() => result.current());
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key });
    expect(tsk.type).toBe(otherType);
  });
});
