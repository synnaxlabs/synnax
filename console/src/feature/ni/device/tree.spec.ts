// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  findTreeRow,
  HOST_ID,
  renderLiveTree,
} from "@/platform/tree/testutil";

const client = createTestClient();

describe("NI device tree item", async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const renderDeviceTree = async (): Promise<string> => {
    const name = uniqueName("ni_dev");
    await client.ni.devices.create({
      rack: rack.key,
      name,
      resourceName: id.create(),
      model: "PXIe-4300",
    });
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: {
        ...createHostItems((c) => [
          Tree.createSource(
            c.ni.devices,
            { where: { rack: { eq: rack.key } } },
            Tree.toResource("ni_device"),
          ),
        ]),
        ...NI.Device.TREE_ITEMS,
      },
    });
    return name;
  };

  it("should list the device on the rack", async () => {
    const name = await renderDeviceTree();
    expect(await findTreeRow(name)).not.toBeNull();
  });

  it("should show a rename from another client", async () => {
    const name = await renderDeviceTree();
    await findTreeRow(name);
    const [dev] = await client.ni.devices.retrieve({
      where: { name: { eq: name } },
    });
    const renamed = uniqueName("renamed");
    await createTestClient().ni.devices.update(dev.key, {
      ...ni.device.configurationZ.parse(dev),
      group: "configuration",
      name: renamed,
    });
    await waitFor(async () => expect(await findTreeRow(renamed)).not.toBeNull());
  });

  it("should preview a dragged device with its type name", () => {
    const key = uuid.create();
    render(
      NI.Device.TREE_ITEMS.ni_device.haulPreview({
        resource: createResource(ni.device.resourceID(key), "Device A"),
        client,
        count: 1,
        resolveColor: () => undefined,
      }),
    );
    expect(screen.getByText("Device A")).toBeTruthy();
    expect(screen.getByText("NI device")).toBeTruthy();
  });
});
