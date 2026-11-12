// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  device,
  group,
  ni,
  NotFoundError,
  rack,
  type resource,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { binary } from "@synnaxlabs/x";
import { fireEvent, type RenderOptions, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { SERVER_CERTIFICATE } from "@/feature/opcua/testutil";
import { Rack } from "@/feature/rack";
import { answerNextCommand } from "@/platform/task/testutil";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  expandTreeRow,
  findModalButton,
  findTreeRow,
  HOST_ID,
  renderLiveTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import { assertDefined, createTestClientWithGrants, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Rack.TREE_ITEMS.rack;

const createRack = async () => await client.racks.create({ name: uniqueName("rack") });

const createNIRackWithScanner = async () => {
  const r = await client.racks.create({
    name: uniqueName("ni_rack"),
    integrations: ["ni"],
  });
  const scanTask = await r.createTask(
    {
      name: uniqueName("ni_scanner"),
      type: NI.Task.SCAN_TYPE,
      config: ni.scanConfigZ.parse({}),
    },
    NI.Task.SCAN_SCHEMAS,
  );
  return { rack: r, scanTask };
};

const createOPCUARackWithScanner = async () => {
  const r = await client.racks.create({
    name: uniqueName("opcua_rack"),
    integrations: [OPCUA.Device.INTEGRATION_KEY],
  });
  const scanTask = await r.createTask({
    name: uniqueName("opcua_scanner"),
    type: OPCUA.Task.SCAN_TYPE,
    config: {},
  });
  return { rack: r, scanTask };
};

const rackResource = (key: number, name: string) =>
  createResource(rack.resourceID(key), name);

const renderMenu = async (
  racks: { key: number; name: string }[],
  as: Client = client,
  onCaughtError?: RenderOptions["onCaughtError"],
) => {
  assertDefined(Item.ContextMenu);
  return await renderTreeContextMenu(Item.ContextMenu, {
    client: as,
    resources: racks.map((r) => rackResource(r.key, r.name)),
    onCaughtError,
  });
};

/**
 * A client that may rename racks and read everything the menu touches, differing only
 * in the type it may create.
 */
const createRackWriter = async (creatable: resource.ID) =>
  await createTestClientWithGrants(client, {
    retrieve: [rack.TYPE_RESOURCE_ID, task.TYPE_RESOURCE_ID, arc.TYPE_RESOURCE_ID],
    update: [rack.TYPE_RESOURCE_ID],
    create: [creatable],
  });

describe("rack tree item", () => {
  describe("drop", () => {
    const root = device.TYPE_RESOURCE_ID;
    const destination = createResource(rack.resourceID(1), "rack");
    const canDrop = (id: resource.ID, ancestors: resource.ID[]): boolean =>
      Item.canDrop({
        source: { key: "tree", type: PTree.HAUL_TYPE },
        items: [PTree.createHaulItem(`${id.type}:${id.key}`)],
        root,
        destination,
        ancestors: [root],
        moved: [{ id, ancestors }],
      });

    it("should accept a device or group of its own rack", () => {
      expect(canDrop(device.resourceID("d"), [rack.resourceID(1), root])).toBe(true);
      expect(canDrop(group.resourceID("g"), [rack.resourceID(1), root])).toBe(true);
    });

    it("should refuse a device or group from another rack", () => {
      expect(canDrop(device.resourceID("d"), [rack.resourceID(2), root])).toBe(false);
      expect(canDrop(group.resourceID("g"), [rack.resourceID(2), root])).toBe(false);
    });

    it("should refuse a device under another device", () => {
      expect(
        canDrop(device.resourceID("d"), [
          device.resourceID("p"),
          rack.resourceID(1),
          root,
        ]),
      ).toBe(false);
    });

    it("should refuse a rack", () => {
      expect(canDrop(rack.resourceID(2), [root])).toBe(false);
    });
  });

  it("should expose rename, arc creation, and delete for a single rack", async () => {
    const r = await createRack();
    await renderMenu([r]);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Create Arc automation")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
  });

  it("should offer Reload Console", async () => {
    const r = await createRack();
    await renderMenu([r]);
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should hide single-selection actions for a multi-rack selection", async () => {
    const a = await createRack();
    const b = await createRack();
    await renderMenu([a, b]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Create Arc automation")).toBeNull();
    expect(screen.queryByText("Copy properties")).toBeNull();
  });

  it("should delete the rack on the Core after confirmation", async () => {
    const r = await createRack();
    // The menu outlives the rack it deletes, and its items throw the deletion.
    await renderMenu([r], client, () => {});
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${r.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(async () => {
      await expect(client.racks.retrieve(r.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should open the arc creation modal from the context menu", async () => {
    const r = await createRack();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Create Arc automation"));
    expect(await screen.findByText("Create Arc automation")).toBeTruthy();
  });

  it("should open the device picker scoped to the rack", async () => {
    const r = await createRack();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Connect device"));
    expect(await screen.findByText("Connect by address")).toBeTruthy();
    expect(
      (await screen.findAllByText(`${r.name} is not running`)).length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText(/^No running Driver has/)).toBeNull();
  });

  it("should hide the toggle scanner item for a rack without the NI integration", async () => {
    const r = await createRack();
    await renderMenu([r]);
    await screen.findByText("Rename");
    expect(screen.queryByText("Toggle NI device scanner")).toBeNull();
  });

  it("should hide the OPC UA Driver certificate for a rack without the OPC UA integration", async () => {
    const r = await createRack();
    await renderMenu([r]);
    await screen.findByText("Rename");
    expect(screen.queryByText("Show OPC UA Driver certificate")).toBeNull();
  });

  it("should open the OPC UA Driver certificate of the rack", async () => {
    const { rack: r, scanTask } = await createOPCUARackWithScanner();
    const { answered } = await answerNextCommand(client, scanTask, {
      type: OPCUA.Task.GET_CERTIFICATE_COMMAND_TYPE,
      data: { current: SERVER_CERTIFICATE },
    });
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Show OPC UA Driver certificate"));
    await answered;
    expect(
      await screen.findByText(
        "FF:65:16:FC:E0:22:F9:7F:23:B6:F6:79:17:9D:B7:FE:68:CB:2D:9C",
      ),
    ).toBeTruthy();
  });

  it("should toggle the NI scanner's disabled flag from the rack context menu", async () => {
    const { rack: r, scanTask } = await createNIRackWithScanner();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Toggle NI device scanner"));
    await waitFor(async () => {
      const after = await client.tasks.retrieve({
        key: scanTask.key,
        schemas: NI.Task.SCAN_SCHEMAS,
      });
      expect(after.config.disabled).toBe(!scanTask.config.disabled);
    });
  });
});

describe("rack children", () => {
  it("should list only the devices that sit directly on the rack", async () => {
    const r = await createRack();
    const make = async (name: string, parent?: device.Key) =>
      await client.devices.create({
        key: uniqueName(name),
        name: uniqueName(name),
        rack: r.key,
        location: name,
        make: "make",
        model: "model",
        properties: {},
        parent,
      });
    const chassis = await make("chassis");
    const module = await make("module", chassis.key);
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: {
        ...Rack.TREE_ITEMS,
        ...Device.TREE_ITEMS,
        ...createHostItems((c) => [
          Tree.createSource(c.racks, { key: [r.key] }, (rk) =>
            Tree.toResource("rack")(rk.payload),
          ),
        ]),
      },
    });
    await findTreeRow(r.name);
    expandTreeRow(r.name);
    await findTreeRow(chassis.name);
    expect(screen.queryByText(module.name)).toBeNull();
  });
});

describe("copy properties", () => {
  it("should copy the stored rack as JSON", async () => {
    const writeText = stubClipboardWriteText();
    const r = await createRack();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Copy properties"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(binary.JSON_CODEC.encodeString(r.payload)),
    );
  });
});

describe("permission to write the rack", () => {
  it("should withhold rename, arc creation, and delete from a viewer", async () => {
    const r = await createRack();
    await renderMenu([r], await roles.get("Viewer"));
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Create Arc automation")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should withhold the OPC UA Driver certificate from a viewer", async () => {
    const { rack: r } = await createOPCUARackWithScanner();
    await renderMenu([r], await roles.get("Viewer"));
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Show OPC UA Driver certificate")).toBeNull();
  });

  it("should withhold the scanner toggle from a subject who cannot create tasks", async () => {
    const { rack: r } = await createNIRackWithScanner();
    await renderMenu([r], await createRackWriter(arc.TYPE_RESOURCE_ID));
    expect(await screen.findByText("Create Arc automation")).toBeTruthy();
    expect(screen.queryByText("Toggle NI device scanner")).toBeNull();
  });

  it("should withhold Connect device from a subject who cannot create devices", async () => {
    const r = await createRack();
    await renderMenu([r], await createRackWriter(arc.TYPE_RESOURCE_ID));
    expect(await screen.findByText("Create Arc automation")).toBeTruthy();
    expect(screen.queryByText("Connect device")).toBeNull();
  });

  it("should offer Connect device to a subject who may create devices", async () => {
    const r = await createRack();
    await renderMenu([r], await createRackWriter(device.TYPE_RESOURCE_ID));
    expect(await screen.findByText("Connect device")).toBeTruthy();
  });

  it("should offer the scanner toggle to a subject who may create tasks", async () => {
    const { rack: r } = await createNIRackWithScanner();
    await renderMenu([r], await createRackWriter(task.TYPE_RESOURCE_ID));
    expect(await screen.findByText("Toggle NI device scanner")).toBeTruthy();
  });
});
