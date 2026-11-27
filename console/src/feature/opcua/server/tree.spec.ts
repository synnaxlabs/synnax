// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { opcua } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  findTreeRow,
  HOST_ID,
  renderLiveTree,
} from "@/platform/tree/testutil";

const client = createTestClient();

describe("OPC UA server tree item", async () => {
  const rack = await client.racks.create({ name: uniqueName("opcua_rack") });
  const renderServerTree = async (): Promise<string> => {
    const name = uniqueName("opcua_server");
    await client.opcua.servers.create({
      rack: rack.key,
      name,
      connection: { endpoint: "opc.tcp://localhost:4840" },
    });
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: {
        ...createHostItems((c) => [
          Tree.createSource(
            c.opcua.servers,
            { where: { rack: { eq: rack.key } } },
            Tree.toResource("opcua_server"),
          ),
        ]),
        ...OPCUA.Server.TREE_ITEMS,
      },
    });
    return name;
  };

  it("should list the server on the rack", async () => {
    const name = await renderServerTree();
    expect(await findTreeRow(name)).not.toBeNull();
  });

  it("should show a rename from another client", async () => {
    const name = await renderServerTree();
    await findTreeRow(name);
    const [server] = await client.opcua.servers.retrieve({
      where: { name: { eq: name } },
    });
    const renamed = uniqueName("renamed");
    await createTestClient().opcua.servers.update(server.key, {
      ...opcua.server.identityZ.parse(server),
      group: "identity",
      name: renamed,
    });
    await waitFor(async () => expect(await findTreeRow(renamed)).not.toBeNull());
  });

  it("should preview a dragged server with its type name", () => {
    const key = uuid.create();
    render(
      OPCUA.Server.TREE_ITEMS.opcua_server.haulPreview({
        resource: createResource(opcua.server.resourceID(key), "Server A"),
        client,
        count: 1,
        resolveColor: () => undefined,
      }),
    );
    expect(screen.getByText("Server A")).toBeTruthy();
    expect(screen.getByText("OPC UA server")).toBeTruthy();
  });
});
