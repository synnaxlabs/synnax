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
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { createDeviceResource } from "@/platform/device/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

const renderItems = async () => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  store.dispatch(Session.Project.select(proj.key));
  const resource = createDeviceResource({
    key: id.create(),
    name: "lj-dev",
    configured: true,
  });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [resource.id] }),
    state: createState([resource]),
  };
  render(
    <PMenu.Menu>
      <LabJack.Device.ContextMenuItems {...props} />
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, key: resource.id.key };
};

describe("LabJack device ContextMenuItems", () => {
  it("should create a read task draft bound to the device", async () => {
    const { store, key } = await renderItems();
    fireEvent.click(await screen.findByText("Create read task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(LabJack.Task.READ_TYPE);
    expect(created.config).toMatchObject({ device: key });
  });

  it("should create a write task draft bound to the device", async () => {
    const { store, key } = await renderItems();
    fireEvent.click(await screen.findByText("Create write task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(LabJack.Task.WRITE_TYPE);
    expect(created.config).toMatchObject({ device: key });
  });
});
