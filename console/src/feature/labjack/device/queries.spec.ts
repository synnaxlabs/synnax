// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type labjack, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

describe("typed LabJack device queries", async () => {
  const rack = await client.racks.create({ name: uniqueName("labjack_rack") });
  let serial = rack.key * 10;
  const create = async (): Promise<labjack.device.Device> =>
    await client.labjack.devices.create({
      rack: rack.key,
      name: uniqueName("labjack_dev"),
      serial: (serial += 1),
      connection: "LJM_ctUSB",
      model: "t7",
      ports: { model: "t7", mux80Attached: true },
    });

  it("should retrieve the device with its status", async () => {
    const dev = await create();
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = renderHook(() => LabJack.Device.useRetrieve({ key: dev.key }), {
      wrapper,
    });
    await waitFor(() => expect(result.current.data?.key).toBe(dev.key));
    expect(result.current.data?.status).toBeDefined();
  });

  it("should rename the device and keep its other fields", async () => {
    const dev = await create();
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = renderHook(() => LabJack.Device.useRename(), { wrapper });
    await act(async () => {
      await result.current.updateAsync({ key: dev.key, name: "Bench T7" });
    });
    const got = await client.labjack.devices.retrieve({ key: dev.key });
    expect(got.name).toBe("Bench T7");
    expect(got.ports).toEqual(dev.ports);
    expect(got.serial).toBe(dev.serial);
  });

  it("should delete the device", async () => {
    const dev = await create();
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = renderHook(() => LabJack.Device.useDelete(), { wrapper });
    await act(async () => {
      await result.current.updateAsync(dev.key);
    });
    await expect(client.labjack.devices.retrieve({ key: dev.key })).rejects.toThrow(
      NotFoundError,
    );
  });
});
