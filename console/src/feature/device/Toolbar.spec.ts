// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, rack as rackClient } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { Group } from "@/feature/group";
import { Rack } from "@/feature/rack";
import { createTestDevice } from "@/platform/device/testutil";
import {
  findTreeRow,
  itemName,
  renderToolbar,
  searchTree,
} from "@/platform/tree/testutil";
import {
  assertDefined,
  createTestClientWithGrants,
  getIconButton,
  renderHookWithConsole,
  uniqueName,
  uniqueToken,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("device/Toolbar", () => {
  it("lists a created device under its rack in the devices tree", async () => {
    const dev = await createTestDevice(client);
    const rack = await client.racks.retrieve(dev.rack);
    await renderToolbar(Device.TOOLBAR.content, {
      client,
      items: { ...Rack.TREE_ITEMS, ...Device.TREE_ITEMS },
    });
    await screen.findByText("Devices");
    fireEvent.click(await findTreeRow(rack.name));
    expect(await findTreeRow(dev.name)).toBeTruthy();
  });
});

const renderDeviceToolbar = async (): Promise<void> => {
  await renderToolbar(Device.TOOLBAR.content, {
    client,
    items: {
      ...Rack.TREE_ITEMS,
      ...Device.TREE_ITEMS,
      ...Group.createTreeItems([Rack.GROUP_MEMBER, Device.GROUP_MEMBER]),
    },
  });
  await screen.findByText("Devices");
};

const createDevice = async (
  overrides: Partial<device.New> = {},
): Promise<device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await client.devices.create({
    key: id.create(),
    name: uniqueName("device"),
    rack: rack.key,
    location: "dev1",
    make: "make",
    model: "model",
    properties: {},
    ...overrides,
  });
};

/** Expects each name to show one level below the name before it. */
const expectNested = async (names: string[]): Promise<void> => {
  const levels: number[] = [];
  for (const name of names) {
    const row = await findTreeRow(itemName(name));
    levels.push(Number(row.getAttribute("aria-level")));
  }
  expect(levels.slice(1)).toEqual(levels.slice(0, -1).map((l) => l + 1));
};

describe("device/Toolbar search", () => {
  it("should find a device by name and show its rack", async () => {
    const tok = uniqueToken();
    const dev = await createDevice({ name: `${tok}_pump` });
    const rack = await client.racks.retrieve(dev.rack);
    await renderDeviceToolbar();
    searchTree(tok);
    const row = await findTreeRow(itemName(dev.name));
    expect(row.querySelector("mark")?.textContent).toBe(tok);
    expect(row.textContent).toContain(rack.name);
  });

  it("should find a device by its location", async () => {
    const tok = uniqueToken();
    const dev = await createDevice({ location: `${tok}.lab.local:502` });
    await renderDeviceToolbar();
    searchTree(tok);
    expect(await findTreeRow(itemName(dev.name))).toBeTruthy();
  });

  it("should find a device by its make or model", async () => {
    const tok = uniqueToken();
    const byMake = await createDevice({ make: `${tok}_make` });
    const byModel = await createDevice({ model: `${tok}_model` });
    await renderDeviceToolbar();
    searchTree(tok);
    expect(await findTreeRow(itemName(byMake.name))).toBeTruthy();
    expect(await findTreeRow(itemName(byModel.name))).toBeTruthy();
  });

  it("should show the chassis that holds a found module", async () => {
    const tok = uniqueToken();
    const chassis = await createDevice();
    const mod = await createDevice({
      name: `${tok}_module`,
      rack: chassis.rack,
      parent: chassis.key,
    });
    await renderDeviceToolbar();
    searchTree(tok);
    expect((await findTreeRow(itemName(mod.name))).textContent).toContain(chassis.name);
  });

  it("should show the group of a found device", async () => {
    const tok = uniqueToken();
    const rack = await client.racks.create({ name: uniqueName("rack") });
    const grp = await client.groups.create({
      name: uniqueName("grp"),
      parent: rackClient.resourceID(rack.key),
    });
    const dev = await createDevice({
      name: `${tok}_valve`,
      rack: rack.key,
      group: grp.key,
    });
    await renderDeviceToolbar();
    searchTree(tok);
    expect((await findTreeRow(itemName(dev.name))).textContent).toContain(grp.name);
  });

  it("should show a module two chassis deep in its groups", async () => {
    const tok = uniqueToken();
    const top = await createDevice();
    const mid = await createDevice({ rack: top.rack, parent: top.key });
    const mod = await createDevice({
      name: `${tok}_module`,
      rack: top.rack,
      parent: mid.key,
    });
    const rack = await client.racks.retrieve(top.rack);
    await renderDeviceToolbar();
    searchTree(tok);
    expect((await findTreeRow(itemName(mod.name))).textContent).toContain(mid.name);
    fireEvent.click(getIconButton(document.body, "tree-view"));
    await expectNested([rack.name, top.name, mid.name, mod.name]);
  });

  it("should show a device in a group under a rack in a group", async () => {
    const tok = uniqueToken();
    const outer = await client.groups.create({
      scope: "device",
      name: uniqueName("out"),
    });
    const rack = await client.racks.create({
      name: uniqueName("rack"),
      group: outer.key,
    });
    const inner = await client.groups.create({
      name: uniqueName("in"),
      parent: rackClient.resourceID(rack.key),
    });
    const dev = await createDevice({
      name: `${tok}_valve`,
      rack: rack.key,
      group: inner.key,
    });
    await renderDeviceToolbar();
    searchTree(tok);
    await findTreeRow(itemName(dev.name));
    fireEvent.click(getIconButton(document.body, "tree-view"));
    await expectNested([outer.name, rack.name, inner.name, dev.name]);
  });

  it("should find a rack by name", async () => {
    const tok = uniqueToken();
    const rack = await client.racks.create({ name: `${tok}_stand` });
    await renderDeviceToolbar();
    searchTree(tok);
    expect(await findTreeRow(itemName(rack.name))).toBeTruthy();
  });
});

describe("device toolbar permissions", () => {
  it("should hide the toolbar from a subject who cannot read devices", async () => {
    const denied = await createTestClientWithGrants(client);
    assertDefined(Device.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Device.TOOLBAR.useVisible, {
      client: denied,
    });
    await waitFor(() => expect(result.current).toBe(false));
  });

  it("should offer the toolbar to a viewer, who may read devices", async () => {
    assertDefined(Device.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Device.TOOLBAR.useVisible, {
      client: await roles.get("Viewer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });
});
