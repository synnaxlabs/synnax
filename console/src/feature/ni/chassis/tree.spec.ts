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

describe("NI chassis tree item", async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const renderChassisTree = async (): Promise<string> => {
    const name = uniqueName("ni_chassis");
    await client.ni.chassis.create({
      rack: rack.key,
      name,
      resourceName: id.create(),
      model: "cDAQ-9178",
    });
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: {
        ...createHostItems((c) => [
          Tree.createSource(
            c.ni.chassis,
            { where: { rack: { eq: rack.key } } },
            Tree.toResource("ni_chassis"),
          ),
        ]),
        ...NI.Chassis.TREE_ITEMS,
      },
    });
    return name;
  };

  it("should list the chassis on the rack", async () => {
    const name = await renderChassisTree();
    expect(await findTreeRow(name)).not.toBeNull();
  });

  it("should show a rename from another client", async () => {
    const name = await renderChassisTree();
    await findTreeRow(name);
    const [chassis] = await client.ni.chassis.retrieve({
      where: { name: { eq: name } },
    });
    const renamed = uniqueName("renamed");
    await createTestClient().ni.chassis.update(chassis.key, {
      group: "configuration",
      name: renamed,
    });
    await waitFor(async () => expect(await findTreeRow(renamed)).not.toBeNull());
  });

  it("should preview a dragged chassis with its type name", () => {
    const key = uuid.create();
    render(
      NI.Chassis.TREE_ITEMS.ni_chassis.haulPreview({
        resource: createResource(ni.chassis.resourceID(key), "Chassis A"),
        client,
        count: 1,
        resolveColor: () => undefined,
      }),
    );
    expect(screen.getByText("Chassis A")).toBeTruthy();
    expect(screen.getByText("NI chassis")).toBeTruthy();
  });
});
