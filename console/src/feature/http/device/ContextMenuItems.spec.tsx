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
  it("should create read and write task drafts bound to the device", async () => {
    const { store, deviceKey } = await renderContextMenuItems(true);
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

  it("should open the connect modal from the edit connection item", async () => {
    await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Edit connection"));
    await screen.findByText("Server");
    expect(screen.getByPlaceholderText("www.example.com")).toBeTruthy();
  });
});
