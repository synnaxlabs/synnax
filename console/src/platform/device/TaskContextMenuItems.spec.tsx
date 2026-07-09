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
import { type SeededPanel, seedSelectedPanel } from "@/platform/task/testutil";
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
  const { store, wrapper } = await renderMenuItem(
    <Device.TaskContextMenuItems
      onConfigure={onConfigure}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
      taskContextMenuItemConfigs={configs}
    />,
    { client: itemClient },
  );
  let seeded: SeededPanel | null = null;
  if (itemClient != null) seeded = await seedSelectedPanel(wrapper, store, itemClient);
  return { onConfigure, key: resource.id.key, store, seeded };
};

const findViewTab = (
  seeded: SeededPanel | null,
  type: string,
): panel.TabView | undefined => {
  assertDefined(seeded, "no panel was seeded");
  const doc = seeded.fluxStore.panels.get(seeded.panelKey);
  if (doc?.root.variant !== "leaf") return undefined;
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
    const { onConfigure, key, seeded } = await setup(false);
    await waitFor(() => expect(screen.getByText("Create Read Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Read Task"));
    expect(onConfigure).toHaveBeenCalledWith(key);
    await waitFor(() => {
      const tab = findViewTab(seeded, "test_read");
      assertDefined(tab, "read task view tab was not opened");
      expect(tab.args).toEqual({ deviceKey: key });
    });
  });

  it("should open the clicked config's view without configuring an already-configured device", async () => {
    const { onConfigure, key, seeded } = await setup(true);
    await waitFor(() => expect(screen.getByText("Create Write Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Write Task"));
    expect(onConfigure).not.toHaveBeenCalled();
    await waitFor(() => {
      const tab = findViewTab(seeded, "test_write");
      assertDefined(tab, "write task view tab was not opened");
      expect(tab.args).toEqual({ deviceKey: key });
    });
    expect(findViewTab(seeded, "test_read")).toBeUndefined();
  });
});
