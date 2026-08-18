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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createDeviceResource, createTestDevice } from "@/platform/device/testutil";
import { Errors } from "@/platform/errors";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderSuspended,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const TASK_LABELS = [
  "Create analog read task",
  "Create analog write task",
  "Create counter read task",
  "Create digital read task",
  "Create digital write task",
];

const renderContextMenu = async () => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  store.dispatch(Session.Project.select(proj.key));
  const dev = await createTestDevice(client, { name: uniqueName("ni_dev") });
  const resource = createDeviceResource({ ...dev, configured: true });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [resource.id] }),
    state: createState([resource]),
  };
  await renderSuspended(
    <PMenu.Menu>
      <Errors.SuspenseBoundary loading={null}>
        <NI.Device.ContextMenuItems {...props} />
      </Errors.SuspenseBoundary>
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, deviceKey: dev.key };
};

describe("device ontology context menu", () => {
  it("should offer every NI task type and create the clicked one bound to the device", async () => {
    const { store, deviceKey } = await renderContextMenu();
    for (const label of TASK_LABELS)
      await waitFor(() => expect(screen.getByText(label)).toBeTruthy());
    fireEvent.click(screen.getByText("Create digital write task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(NI.Task.DIGITAL_WRITE_TYPE);
    expect(created.config).toMatchObject({ device: deviceKey });
  });

  it("should create an analog read draft with a channel on the device", async () => {
    const { store, deviceKey } = await renderContextMenu();
    fireEvent.click(await screen.findByText("Create analog read task"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(NI.Task.ANALOG_READ_TYPE);
    expect(created.config).toMatchObject({ channels: [{ device: deviceKey }] });
  });
});
