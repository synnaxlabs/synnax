// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { useSetDataSaving } from "@/hardware/task/useSetDataSaving";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("useSetDataSaving", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let rack: { key: number; createTask: (...args: any[]) => Promise<any> };

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    rack = await client.racks.create({ name: `test-rack-${id.create()}` });
  });

  it("should enable data saving on a task with dataSaving in config", async () => {
    const t = await rack.createTask({
      name: "read-task",
      type: "testReadType",
      config: { dataSaving: false, sampleRate: 100 },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: t.key, dataSaving: true });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve({ key: t.key });
    expect(updated.config).toEqual({ dataSaving: true, sampleRate: 100 });
  });

  it("should disable data saving on a task with dataSaving in config", async () => {
    const t = await rack.createTask({
      name: "read-task",
      type: "testReadType",
      config: { dataSaving: true, sampleRate: 100 },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: t.key, dataSaving: false });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve({ key: t.key });
    expect(updated.config).toEqual({ dataSaving: false, sampleRate: 100 });
  });

  it("should skip tasks without dataSaving in config", async () => {
    const writeTask = await rack.createTask({
      name: "write-task",
      type: "testWriteType",
      config: { outputChannel: 42 },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: writeTask.key, dataSaving: true });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve({ key: writeTask.key });
    expect(updated.config).toEqual({ outputChannel: 42 });
  });

  it("should skip tasks already at the desired dataSaving value", async () => {
    const t = await rack.createTask({
      name: "already-enabled",
      type: "testReadType",
      config: { dataSaving: true },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: t.key, dataSaving: true });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve({ key: t.key });
    expect(updated.config).toEqual({ dataSaving: true });
  });
});
