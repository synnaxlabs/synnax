// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Device as PDevice } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, it } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice, renderWithDeviceForm } from "@/platform/device/testutil";

const client = createTestClient();

const useFromConfig = Device.createUseFromConfig(PDevice.useResult);

const DeviceName = () => <span>{useFromConfig()?.name ?? "no device"}</span>;

describe("createUseFromConfig", () => {
  it("should return the device the form's config names", async () => {
    const dev = await createTestDevice(client);
    await renderWithDeviceForm(<DeviceName />, { deviceKey: dev.key, client });
    await screen.findByText(dev.name);
  });

  it("should follow the config from no device to a selected one", async () => {
    const dev = await createTestDevice(client);
    await renderWithDeviceForm(<DeviceName />, {
      deviceKey: "",
      selectableKey: dev.key,
      client,
    });
    await screen.findByText("no device");
    fireEvent.click(screen.getByText("select device"));
    await screen.findByText(dev.name);
  });
});
