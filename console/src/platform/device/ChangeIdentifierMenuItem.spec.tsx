// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import {
  createDeviceEntry,
  createTestDevice,
  renderMenuItem,
} from "@/platform/device/testutil";
import { createSelection } from "@/platform/tree/testutil";

const client = createTestClient();

// Runs the async callback the menu item hands to handleError, as the real handler does.
const runHandler: Status.ErrorHandler = (funcOrExc) => {
  if (typeof funcOrExc === "function") void funcOrExc();
};

const renderItem = async (devices: device.Device[], itemClient: Synnax | null) => {
  const entries = devices.map(createDeviceEntry);
  return await renderMenuItem(
    <Device.ChangeIdentifierMenuItem
      icon="Hardware"
      selection={createSelection({ ids: entries.map((r) => r.id) })}
      handleError={runHandler}
    />,
    { client: itemClient },
  );
};

describe("ChangeIdentifierMenuItem", () => {
  it("should render nothing without update permission", async () => {
    const dev = await createTestDevice(client, { configured: true });
    await renderItem([dev], null);
    expect(screen.queryByText("Change identifier")).toBeNull();
  });

  it("should render nothing when more than one device is selected", async () => {
    const a = await createTestDevice(client, { configured: true });
    const b = await createTestDevice(client, { configured: true });
    const entries = [a, b].map(createDeviceEntry);
    await renderMenuItem(
      <>
        <Device.ChangeIdentifierMenuItem
          icon="Hardware"
          selection={createSelection({ ids: [entries[0].id] })}
          handleError={runHandler}
        />
        <Device.ChangeIdentifierMenuItem
          icon="Hardware"
          selection={createSelection({ ids: entries.map((r) => r.id) })}
          handleError={runHandler}
        />
      </>,
      { client },
    );
    await screen.findByText("Change identifier");
    expect(screen.getAllByText("Change identifier")).toHaveLength(1);
  });

  it("should render nothing when the device is not configured", async () => {
    const dev = await createTestDevice(client, { configured: false });
    const entry = createDeviceEntry(dev);
    const selection = createSelection({ ids: [entry.id] });
    await renderMenuItem(
      <>
        <Device.ConfigureMenuItem onConfigure={() => {}} selection={selection} />
        <Device.ChangeIdentifierMenuItem
          icon="Hardware"
          selection={selection}
          handleError={runHandler}
        />
      </>,
      { client },
    );
    await screen.findByText("Configure");
    expect(screen.queryByText("Change identifier")).toBeNull();
  });

  it("should update the device identifier through the rename modal, prefilled with the current one", async () => {
    const dev = await createTestDevice(client, {
      configured: true,
      properties: { identifier: "old_id" },
    });
    await renderItem([dev], client);
    await waitFor(() => expect(screen.getByText("Change identifier")).toBeTruthy());
    fireEvent.click(screen.getByText("Change identifier"));
    const input = await screen.findByRole("textbox");
    await waitFor(() => expect((input as HTMLInputElement).value).toEqual("old_id"));
    fireEvent.change(input, { target: { value: "new_id" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(async () => {
      const updated = await client.devices.retrieve({ key: dev.key });
      expect(updated.properties.identifier).toEqual("new_id");
    });
  });

  it("should leave the identifier unchanged when the rename prompt is dismissed", async () => {
    const dev = await createTestDevice(client, {
      configured: true,
      properties: { identifier: "old_id" },
    });
    const { modals } = await renderItem([dev], client);
    await waitFor(() => expect(screen.getByText("Change identifier")).toBeTruthy());
    fireEvent.click(screen.getByText("Change identifier"));
    await screen.findByRole("textbox");
    act(() => modals.getState()[0].dismiss());
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
    const unchanged = await client.devices.retrieve({ key: dev.key });
    expect(unchanged.properties.identifier).toEqual("old_id");
  });
});
