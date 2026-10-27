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
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { HTTP } from "@/feature/http";
import { type Device } from "@/platform/device";
import { createDeviceResource } from "@/platform/device/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

const renderContextMenuItems = async () => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  const dev = await client.devices.create({
    key: id.create(),
    name: "http_dev",
    make: "http",
    model: "HTTP server",
    location: "http://localhost",
    rack: rack.key,
    configured: true,
    properties: HTTP.Device.ZERO_PROPERTIES,
  });
  const resource = createDeviceResource({
    key: dev.key,
    name: dev.name,
    configured: true,
  });
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
  });
  store.dispatch(Session.Project.select(proj.key));
  const props: Device.ContextMenuItemsProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [resource.id] }),
    state: createState([resource]),
    onConfigure: vi.fn(),
  };
  render(
    <PMenu.Menu>
      <HTTP.ContextMenuItems {...props} />
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, deviceKey: resource.id.key };
};

describe("HTTP device ContextMenuItems", () => {
  it("should create read and write task drafts bound to the device", async () => {
    const { store, deviceKey } = await renderContextMenuItems();
    fireEvent.click(await screen.findByText("Create read task"));
    const readTab = await resolveFocusedTab(store, client);
    if (readTab.variant !== "resource") throw new Error("expected a resource tab");
    expect(readTab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const read = await client.tasks.retrieve({ key: readTab.resource.key });
    expect(read.type).toBe(HTTP.Task.READ_TYPE);
    expect(read.config).toMatchObject({ device: deviceKey });
    fireEvent.click(screen.getByText("Create write task"));
    const writeTab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "resource" && t.resource.key !== readTab.resource.key,
    );
    if (writeTab.variant !== "resource") throw new Error("expected a resource tab");
    const write = await client.tasks.retrieve({ key: writeTab.resource.key });
    expect(write.type).toBe(HTTP.Task.WRITE_TYPE);
    expect(write.config).toMatchObject({ device: deviceKey });
  });
});
