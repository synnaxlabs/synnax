// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createDeviceResource } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

const renderContextMenuItems = async (configured: boolean) => {
  const resource = createDeviceResource({
    key: id.create(),
    name: "http_dev",
    configured,
  });
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  store.dispatch(Session.Project.select(proj.key));
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [resource.id] }),
    state: createState([resource]),
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
  return { store, deviceKey: resource.id.key };
};

describe("HTTP device ContextMenuItems", () => {
  it("should open the read and write task views carrying the device key", async () => {
    const { store, deviceKey } = await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Create read task"));
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: HTTP.Task.READ_TYPE,
      args: { deviceKey },
    });
    fireEvent.click(screen.getByText("Create write task"));
    expect(
      await resolveFocusedTab(
        store,
        client,
        (t) => t.variant === "view" && t.type === HTTP.Task.WRITE_TYPE,
      ),
    ).toMatchObject({
      variant: "view",
      type: HTTP.Task.WRITE_TYPE,
      args: { deviceKey },
    });
  });

  it("should open the connect modal from the edit connection item", async () => {
    await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Edit connection"));
    await screen.findByText("Server");
    expect(screen.getByPlaceholderText("www.example.com")).toBeTruthy();
  });
});
