// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { PagerDuty } from "@/feature/pagerduty";
import { useSetDataSaving } from "@/feature/task/useSetDataSaving";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

describe("useSetDataSaving", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let rack: { key: number; createTask: (...args: any[]) => Promise<any> };

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    rack = await client.racks.create({ name: `test-rack-${id.create()}` });
  });

  it("should enable data saving on a task with the field in config", async () => {
    const t = await rack.createTask({
      name: "read-task",
      type: NI.Task.ANALOG_READ_TYPE,
      config: { dataSavingDisabled: true, sampleRate: 100 },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: t.key, dataSaving: true });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve(t.key);
    expect(updated.config).toMatchObject({
      dataSavingDisabled: false,
      sampleRate: 100,
    });
  });

  it("should disable data saving on a task with the field in config", async () => {
    const t = await rack.createTask({
      name: "read-task",
      type: NI.Task.ANALOG_READ_TYPE,
      config: { dataSavingDisabled: false, sampleRate: 100 },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: t.key, dataSaving: false });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve(t.key);
    expect(updated.config).toMatchObject({
      dataSavingDisabled: true,
      sampleRate: 100,
    });
  });

  it("should skip tasks without the field in config", async () => {
    const alertTask = await rack.createTask({
      name: "alert-task",
      type: PagerDuty.Task.ALERT_TYPE,
      config: { routingKey: "a".repeat(32) },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: alertTask.key, dataSaving: true });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve(alertTask.key);
    expect(updated.config).not.toHaveProperty("dataSavingDisabled");
  });

  it("should skip tasks already at the desired dataSaving value", async () => {
    const t = await rack.createTask({
      name: "already-enabled",
      type: NI.Task.ANALOG_READ_TYPE,
      config: { dataSavingDisabled: false },
    });

    const { result } = renderHook(() => useSetDataSaving(), { wrapper });

    await act(async () => {
      await result.current.updateAsync({ key: t.key, dataSaving: true });
    });

    await waitFor(() => expect(result.current.variant).toEqual("success"));

    const updated = await client.tasks.retrieve(t.key);
    expect(updated.config).toMatchObject({ dataSavingDisabled: false });
  });
});
