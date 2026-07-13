// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import {
  createDeviceResource,
  createTestDevice,
  renderMenuItem,
} from "@/platform/device/testutil";
import { Task } from "@/platform/task";
import { createSelection } from "@/platform/tree/testutil";
import { Session } from "@/session";

const client = createTestClient();

const configs: Device.TaskContextMenuItemConfig[] = [
  {
    itemKey: "read",
    label: "Create Read Task",
    layout: { ...Task.LAYOUT, type: "test_read", key: "test_read" },
  },
  {
    itemKey: "write",
    label: "Create Write Task",
    layout: { ...Task.LAYOUT, type: "test_write", key: "test_write" },
  },
];

const setup = async (configured: boolean, itemClient: Synnax | null = client) => {
  const onConfigure = vi.fn();
  const dev = await createTestDevice(client, { configured });
  const resource = createDeviceResource(dev);
  const selection = createSelection({ ids: [resource.id] });
  const { store } = await renderMenuItem(
    <>
      <Device.ChangeIdentifierMenuItem
        icon="Hardware"
        selection={selection}
        handleError={() => {}}
      />
      <Device.TaskContextMenuItems
        onConfigure={onConfigure}
        selection={selection}
        taskContextMenuItemConfigs={configs}
      />
    </>,
    { client: itemClient },
  );
  return { onConfigure, key: dev.key, store };
};

describe("TaskContextMenuItems", () => {
  it("should render nothing without task-create permission", async () => {
    await setup(true, null);
    expect(screen.queryByText("Create Read Task")).toBeNull();
    expect(screen.queryByText("Create Write Task")).toBeNull();
  });

  it("should configure an unconfigured device and place the task layout with its key", async () => {
    const { onConfigure, key, store } = await setup(false);
    await waitFor(() => expect(screen.getByText("Create Read Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Read Task"));
    expect(onConfigure).toHaveBeenCalledWith(key);
    const layout = Session.Layout.select(store.getState(), "test_read");
    expect(layout?.type).toEqual("test_read");
    expect(Session.Layout.selectArgs(store.getState(), "test_read")).toEqual({
      deviceKey: key,
    });
  });

  it("should place the clicked config's layout without configuring an already-configured device", async () => {
    const { onConfigure, key, store } = await setup(true);
    // "Change identifier" renders only once the device retrieval resolves
    // configured === true, so it gates the click on the same query the
    // task items read the configured flag from.
    await screen.findByText("Change identifier");
    await waitFor(() => expect(screen.getByText("Create Write Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Write Task"));
    expect(onConfigure).not.toHaveBeenCalled();
    expect(Session.Layout.select(store.getState(), "test_write")?.type).toEqual(
      "test_write",
    );
    expect(Session.Layout.selectArgs(store.getState(), "test_write")).toEqual({
      deviceKey: key,
    });
    expect(Session.Layout.select(store.getState(), "test_read")).toBeUndefined();
  });
});
