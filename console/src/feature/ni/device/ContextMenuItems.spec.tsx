// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
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
  const resource = createDeviceResource({
    key: id.create(),
    name: "ni_dev",
    configured: true,
  });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [resource.id] }),
    state: createState([resource]),
  };
  render(
    <PMenu.Menu>
      <NI.Device.ContextMenuItems {...props} />
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, deviceKey: resource.id.key };
};

describe("device ontology context menu", () => {
  it("should offer every NI task type and open the clicked one with the device key", async () => {
    const { store, deviceKey } = await renderContextMenu();
    for (const label of TASK_LABELS)
      await waitFor(() => expect(screen.getByText(label)).toBeTruthy());
    fireEvent.click(screen.getByText("Create digital write task"));
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: NI.Task.DIGITAL_WRITE_TYPE,
      args: { deviceKey },
    });
  });

  it("should open the analog read view with the device key when clicked", async () => {
    const { store, deviceKey } = await renderContextMenu();
    fireEvent.click(await screen.findByText("Create analog read task"));
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: NI.Task.ANALOG_READ_TYPE,
      args: { deviceKey },
    });
  });
});
