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
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createDeviceEntry, createTestDevice } from "@/platform/device/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

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
  const dev = await createTestDevice(client, { configured: true });
  const entry = createDeviceEntry(dev);
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [entry.id] }),
    state: createState([entry]),
  };
  render(
    <PMenu.Menu>
      <NI.Device.ContextMenuItems {...props} />
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, deviceKey: entry.id.key };
};

describe("device ontology context menu", () => {
  it("should offer every NI task type and place the clicked one with the device key", async () => {
    const { store, deviceKey } = await renderContextMenu();
    for (const label of TASK_LABELS)
      await waitFor(() => expect(screen.getByText(label)).toBeTruthy());
    fireEvent.click(screen.getByText("Create digital write task"));
    await waitFor(() => {
      const placed = Session.Layout.selectByFilter(
        store.getState(),
        (l) => l.type === NI.Task.DIGITAL_WRITE_TYPE,
      );
      if (placed == null) throw new Error("digital write layout not placed");
      expect(Session.Layout.selectArgs(store.getState(), placed.key)).toEqual({
        deviceKey,
      });
    });
  });

  it("should place the analog read layout with the device key when clicked", async () => {
    const { store, deviceKey } = await renderContextMenu();
    fireEvent.click(await screen.findByText("Create analog read task"));
    await waitFor(() => {
      const placed = Session.Layout.selectByFilter(
        store.getState(),
        (l) => l.type === NI.Task.ANALOG_READ_TYPE,
      );
      if (placed == null) throw new Error("analog read layout not placed");
      expect(Session.Layout.selectArgs(store.getState(), placed.key)).toEqual({
        deviceKey,
      });
    });
  });
});
