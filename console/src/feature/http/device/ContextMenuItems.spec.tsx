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

import { HTTP } from "@/feature/http";
import { createDeviceEntry, createTestDevice } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, waitForPlacedLayout } from "@/testutil";

const client = createTestClient();

const renderContextMenuItems = async (configured: boolean) => {
  const dev = await createTestDevice(client, { configured });
  const entry = createDeviceEntry(dev);
  const { wrapper, store } = await createConsoleWrapper({ client });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [entry.id] }),
    state: createState([entry]),
  };
  render(
    <>
      <PMenu.Menu>
        <HTTP.Device.ContextMenuItems {...props} />
      </PMenu.Menu>
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store, deviceKey: entry.id.key };
};

describe("HTTP device ContextMenuItems", () => {
  it("should place the read and write task layouts carrying the device key", async () => {
    const { store, deviceKey } = await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Create read task"));
    const readKey = await waitForPlacedLayout(store, HTTP.Task.READ_TYPE);
    expect(Session.Layout.selectArgs(store.getState(), readKey)).toEqual({
      deviceKey,
    });
    fireEvent.click(screen.getByText("Create write task"));
    const writeKey = await waitForPlacedLayout(store, HTTP.Task.WRITE_TYPE);
    expect(Session.Layout.selectArgs(store.getState(), writeKey)).toEqual({
      deviceKey,
    });
  });

  it("should open the connect modal from the edit connection item", async () => {
    await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Edit connection"));
    await screen.findByText("Server");
    expect(screen.getByPlaceholderText("www.example.com")).toBeTruthy();
  });
});
