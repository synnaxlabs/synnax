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
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { createDeviceEntry, createTestDevice } from "@/platform/device/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, waitForPlacedLayout } from "@/testutil";

const client = createTestClient();

const renderItems = async () => {
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
      <LabJack.Device.ContextMenuItems {...props} />
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, key: entry.id.key };
};

describe("LabJack device ContextMenuItems", () => {
  it("should place the read task layout bound to the device", async () => {
    const { store, key } = await renderItems();
    fireEvent.click(await screen.findByText("Create read task"));
    const layoutKey = await waitForPlacedLayout(store, LabJack.Task.READ_TYPE);
    expect(Session.Layout.selectArgs(store.getState(), layoutKey)).toEqual({
      deviceKey: key,
    });
  });

  it("should place the write task layout bound to the device", async () => {
    const { store, key } = await renderItems();
    fireEvent.click(await screen.findByText("Create write task"));
    const layoutKey = await waitForPlacedLayout(store, LabJack.Task.WRITE_TYPE);
    expect(Session.Layout.selectArgs(store.getState(), layoutKey)).toEqual({
      deviceKey: key,
    });
  });
});
