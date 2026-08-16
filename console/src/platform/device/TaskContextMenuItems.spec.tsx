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
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createDeviceResource, renderMenuItem } from "@/platform/device/testutil";
import { createSelection, createState } from "@/platform/tree/testutil";

const client = createTestClient();

const readCreate = vi.fn();
const writeCreate = vi.fn();

const configs: Device.TaskContextMenuItemConfig[] = [
  { itemKey: "read", label: "Create Read Task", useCreate: () => readCreate },
  { itemKey: "write", label: "Create Write Task", useCreate: () => writeCreate },
];

const setup = async (configured: boolean, itemClient: Synnax | null = client) => {
  const onConfigure = vi.fn();
  const resource = createDeviceResource({ key: id.create(), name: "dev", configured });
  await renderMenuItem(
    <Device.TaskContextMenuItems
      onConfigure={onConfigure}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
      taskContextMenuItemConfigs={configs}
    />,
    { client: itemClient },
  );
  return { onConfigure, key: resource.id.key };
};

describe("TaskContextMenuItems", () => {
  beforeEach(() => {
    readCreate.mockClear();
    writeCreate.mockClear();
  });

  it("should render nothing without task-create permission", async () => {
    await setup(true, null);
    expect(screen.queryByText("Create Read Task")).toBeNull();
    expect(screen.queryByText("Create Write Task")).toBeNull();
  });

  it("should configure an unconfigured device and create the task with its key", async () => {
    const { onConfigure, key } = await setup(false);
    await waitFor(() => expect(screen.getByText("Create Read Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Read Task"));
    expect(onConfigure).toHaveBeenCalledWith(key);
    expect(readCreate).toHaveBeenCalledWith({ deviceKey: key });
    expect(writeCreate).not.toHaveBeenCalled();
  });

  it("should create from the clicked config without configuring an already-configured device", async () => {
    const { onConfigure, key } = await setup(true);
    await waitFor(() => expect(screen.getByText("Create Write Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Write Task"));
    expect(onConfigure).not.toHaveBeenCalled();
    expect(writeCreate).toHaveBeenCalledWith({ deviceKey: key });
    expect(readCreate).not.toHaveBeenCalled();
  });
});
