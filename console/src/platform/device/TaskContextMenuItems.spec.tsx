// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createDeviceResource, renderMenuItem } from "@/platform/device/testutil";
import { type CreatedPanel, createSelectedPanel } from "@/platform/task/testutil";
import { createSelection, createState } from "@/platform/tree/testutil";
import { assertDefined } from "@/testutil";

const client = createTestClient();

const configs: Device.TaskContextMenuItemConfig[] = [
  { itemKey: "read", label: "Create Read Task", type: "test_read" },
  { itemKey: "write", label: "Create Write Task", type: "test_write" },
];

const setup = async (configured: boolean, itemClient: Synnax | null = client) => {
  const onConfigure = vi.fn();
  const resource = createDeviceResource({ key: id.create(), name: "dev", configured });
  const { store } = await renderMenuItem(
    <Device.TaskContextMenuItems
      onConfigure={onConfigure}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
      taskContextMenuItemConfigs={configs}
    />,
    { client: itemClient },
  );
  let created: CreatedPanel | null = null;
  if (itemClient != null) {
    created = await createSelectedPanel(store, itemClient);
    // useOpenTab reads the panel query cache; warm it and keep it subscribed
    // so dispatches stay visible.
    await itemClient.panels.retrieve({ key: created.panelKey });
  }
  return { onConfigure, key: resource.id.key, store, created };
};

const findViewTab = async (
  created: CreatedPanel | null,
  type: string,
): Promise<panel.TabView | undefined> => {
  assertDefined(created, "no panel was created");
  const doc = await client.panels.retrieve({ key: created.panelKey });
  if (doc.root.variant !== "leaf") return undefined;
  return doc.root.tabs.find(
    (t): t is panel.TabView => t.variant === "view" && t.type === type,
  );
};

describe("TaskContextMenuItems", () => {
  it("should render nothing without task-create permission", async () => {
    await setup(true, null);
    expect(screen.queryByText("Create Read Task")).toBeNull();
    expect(screen.queryByText("Create Write Task")).toBeNull();
  });

  it("should configure an unconfigured device and open the task view with its key", async () => {
    const { onConfigure, key, created } = await setup(false);
    await waitFor(() => expect(screen.getByText("Create Read Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Read Task"));
    expect(onConfigure).toHaveBeenCalledWith(key);
    await waitFor(async () => {
      const tab = await findViewTab(created, "test_read");
      assertDefined(tab, "read task view tab was not opened");
      expect(tab.args).toEqual({ deviceKey: key });
    });
  });

  it("should open the clicked config's view without configuring an already-configured device", async () => {
    const { onConfigure, key, created } = await setup(true);
    await waitFor(() => expect(screen.getByText("Create Write Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Write Task"));
    expect(onConfigure).not.toHaveBeenCalled();
    await waitFor(async () => {
      const tab = await findViewTab(created, "test_write");
      assertDefined(tab, "write task view tab was not opened");
      expect(tab.args).toEqual({ deviceKey: key });
    });
    expect(await findViewTab(created, "test_read")).toBeUndefined();
  });
});
