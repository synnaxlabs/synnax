// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Device } from "@/platform/device";
import {
  createDeviceResource,
  createTestDevice,
  renderMenuItem,
} from "@/platform/device/testutil";
import { createSelection } from "@/platform/tree/testutil";

const ADAPTER = Device.createLegacyAdapter({ make: "acme", zeroProperties: {} });
const ConfigureMenuItem = Device.createConfigureMenuItem(ADAPTER);
const ChangeIdentifierMenuItem = Device.createChangeIdentifierMenuItem({
  adapter: ADAPTER,
  icon: "Hardware",
});

const client = createTestClient();
const roles = new RoleClients(client);

const renderItem = async (
  devices: device.Device[],
  itemClient: Synnax,
): Promise<Mock> => {
  const onConfigure = vi.fn();
  const resources = devices.map(createDeviceResource);
  await renderMenuItem(
    <ConfigureMenuItem
      onConfigure={onConfigure}
      selection={createSelection({ ids: resources.map((r) => r.id) })}
    />,
    { client: itemClient },
  );
  return onConfigure;
};

describe("ConfigureMenuItem", () => {
  it("should render nothing without update permission", async () => {
    const dev = await createTestDevice(client, { configured: false });
    await renderItem([dev], await roles.get("Viewer"));
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("should render nothing when more than one device is selected", async () => {
    const a = await createTestDevice(client, { configured: false });
    const b = await createTestDevice(client, { configured: false });
    const resources = [a, b].map(createDeviceResource);
    await renderMenuItem(
      <>
        <ConfigureMenuItem
          onConfigure={vi.fn()}
          selection={createSelection({ ids: [resources[0].id] })}
        />
        <ConfigureMenuItem
          onConfigure={vi.fn()}
          selection={createSelection({ ids: resources.map((r) => r.id) })}
        />
      </>,
      { client },
    );
    await screen.findByText("Configure");
    expect(screen.getAllByText("Configure")).toHaveLength(1);
  });

  it("should render nothing when the device is already configured", async () => {
    const dev = await createTestDevice(client, { configured: true });
    const resource = createDeviceResource(dev);
    const selection = createSelection({ ids: [resource.id] });
    await renderMenuItem(
      <>
        <ChangeIdentifierMenuItem selection={selection} handleError={() => {}} />
        <ConfigureMenuItem onConfigure={vi.fn()} selection={selection} />
      </>,
      { client },
    );
    await screen.findByText("Change identifier");
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("should invoke onConfigure with the device key when clicked", async () => {
    const dev = await createTestDevice(client, { configured: false });
    const onConfigure = await renderItem([dev], client);
    await waitFor(() => expect(screen.getByText("Configure")).toBeTruthy());
    fireEvent.click(screen.getByText("Configure"));
    expect(onConfigure).toHaveBeenCalledWith(dev.key);
  });
});
