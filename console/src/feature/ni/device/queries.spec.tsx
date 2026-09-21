// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createNIDevice } from "@/feature/ni/task/testutil";
import { renderWithDeviceForm } from "@/platform/device/testutil";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";

const client = createTestClient();

const DeviceName = () => <span>{NI.Device.useFromConfig()?.name ?? "no device"}</span>;

describe("use", () => {
  it("should retrieve the device with its typed properties", async () => {
    const dev = await createNIDevice(client);
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = await renderHookSuspended(
      () => NI.Device.use({ key: dev.key }),
      { wrapper },
    );
    await waitFor(() =>
      expect(result.current?.properties.identifier).toBe(dev.properties.identifier),
    );
  });
});

describe("useResult", () => {
  it("should resolve the device without suspending", async () => {
    const dev = await createNIDevice(client);
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const { result } = renderHook(() => NI.Device.useResult({ key: dev.key }), {
      wrapper,
    });
    await waitFor(() => expect(result.current.data?.key).toBe(dev.key));
    expect(result.current.variant).toBe("success");
  });
});

describe("useFromConfig", () => {
  it("should return the device the form's config names", async () => {
    const dev = await createNIDevice(client);
    await renderWithDeviceForm(<DeviceName />, { deviceKey: dev.key, client });
    await screen.findByText(dev.name);
  });
});

describe("useByKeys", () => {
  it("should retrieve the devices the keys name once there are keys", async () => {
    const first = await createNIDevice(client);
    const second = await createNIDevice(client);
    const wrapper = await createAsyncSynnaxWrapper({ client });
    const none: string[] = [];
    const { result, rerender } = renderHook(
      ({ keys }: { keys: string[] }) => NI.Device.useByKeys(keys),
      { wrapper, initialProps: { keys: none } },
    );
    await act(async () => {});
    expect(result.current).toBeUndefined();
    rerender({ keys: [first.key, second.key] });
    await waitFor(() =>
      expect(result.current?.map(({ key }) => key).sort()).toEqual(
        [first.key, second.key].sort(),
      ),
    );
  });
});
