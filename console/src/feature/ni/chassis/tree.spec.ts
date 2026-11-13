// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  createHostItems,
  findTreeRow,
  HOST_ID,
  renderLiveTree,
} from "@/platform/tree/testutil";
import { uniqueName } from "@/testutil";

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
        ...createHostItems((c) => [NI.Chassis.createRackSource(c, rack.key)]),
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
});
