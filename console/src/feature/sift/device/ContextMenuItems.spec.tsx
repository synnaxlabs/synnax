// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import { createDeviceResource, renderMenuItem } from "@/platform/device/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createTestStore, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

const renderItems = async () => {
  const dev = await createSiftDevice(client);
  const resource = createDeviceResource(dev);
  const onConfigure = vi.fn();
  const handle = await renderMenuItem(
    <Sift.Device.ContextMenuItems
      {...createBaseProps({ client, store: await createTestStore() })}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
      onConfigure={onConfigure}
    />,
    { client },
  );
  return { ...handle, dev, onConfigure };
};

describe("Sift.Device.ContextMenuItems", () => {
  it("should open the connection of the selected device for editing", async () => {
    const { dev, onConfigure } = await renderItems();
    fireEvent.click(await screen.findByText("Edit connection"));
    expect(onConfigure).toHaveBeenCalledWith(dev.key);
  });

  it("should create a stream task draft bound to the device", async () => {
    const { store, dev } = await renderItems();
    const proj = await client.projects.create({ name: uniqueName("proj") });
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText("Create stream task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_RESOURCE_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(Sift.Task.STREAM_TYPE);
    expect(created.config).toMatchObject({ device: dev.key });
  });
});
