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

import { MQTT } from "@/feature/mqtt";
import { createDeviceResource } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  clickAndSettle,
  createConsoleWrapper,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const renderContextMenuItems = async (configured: boolean) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  const dev = await client.devices.create({
    key: id.create(),
    name: "mqtt_dev",
    make: MQTT.Device.MAKE,
    model: "MQTT broker",
    location: "localhost",
    rack: rack.key,
    configured,
    properties: MQTT.Device.ZERO_PROPERTIES,
  });
  const resource = createDeviceResource({ key: dev.key, name: dev.name, configured });
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
        <MQTT.Device.ContextMenuItems {...props} />
      </PMenu.Menu>
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store, deviceKey: resource.id.key };
};

describe("MQTT device ContextMenuItems", () => {
  it("should create read and write task drafts bound to the device", async () => {
    const { store, deviceKey } = await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Create read task"));
    const readTab = await resolveFocusedTab(store, client);
    if (readTab.variant !== "resource") throw new Error("expected a resource tab");
    expect(readTab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const read = await client.tasks.retrieve({ key: readTab.resource.key });
    expect(read.type).toBe(MQTT.Task.READ_TYPE);
    expect(read.config).toMatchObject({ device: deviceKey });
    fireEvent.click(screen.getByText("Create write task"));
    const writeTab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "resource" && t.resource.key !== readTab.resource.key,
    );
    if (writeTab.variant !== "resource") throw new Error("expected a resource tab");
    const write = await client.tasks.retrieve({ key: writeTab.resource.key });
    expect(write.type).toBe(MQTT.Task.WRITE_TYPE);
    expect(write.config).toMatchObject({ device: deviceKey });
  });

  it("should create an edge node draft bound to the device", async () => {
    const { store, deviceKey } = await renderContextMenuItems(true);
    fireEvent.click(await screen.findByText("Create Sparkplug edge node"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    const edge = await client.tasks.retrieve({ key: tab.resource.key });
    expect(edge.type).toBe(MQTT.Task.EDGE_TYPE);
    expect(edge.config).toMatchObject({ device: deviceKey });
  });

  it("should open the connect modal from the edit connection item", async () => {
    await renderContextMenuItems(true);
    await clickAndSettle("Edit connection");
    await screen.findByText("Broker");
    expect(screen.getByPlaceholderText("broker.example.com")).toBeTruthy();
  });
});
