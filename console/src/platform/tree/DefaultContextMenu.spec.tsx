// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  channel,
  type resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  type BuiltInRole,
  createTestClient,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { act, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DefaultContextMenu } from "@/platform/tree/DefaultContextMenu";
import { createResource, createState } from "@/platform/tree/testutil";
import { createConsoleWrapper, renderSuspended, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const renderMenus = async (
  as: Client,
  roots: resource.ID[] = [channel.TYPE_RESOURCE_ID],
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  await renderSuspended(
    <>
      {roots.map((root) => (
        <DefaultContextMenu
          key={root.type}
          root={root}
          state={createState([createResource(root, uniqueName("root"))])}
        />
      ))}
    </>,
    { wrapper },
  );
};

describe("Tree.DefaultContextMenu", () => {
  it("should offer the new group item to an engineer", async () => {
    await renderMenus(await roles.get("Engineer"));
    expect(await screen.findByText("New group")).toBeTruthy();
  });

  it.each<BuiltInRole>(["Viewer", "Operator"])(
    "should withhold the new group item from a %s",
    async (role) => {
      await renderMenus(await roles.get(role));
      expect(await screen.findByText("Reload Console")).toBeTruthy();
      expect(screen.queryByText("New group")).toBeNull();
    },
  );

  it("should withhold the new group item from a tree that holds no groups", async () => {
    await renderMenus(await roles.get("Engineer"), [
      channel.TYPE_RESOURCE_ID,
      access.role.TYPE_RESOURCE_ID,
    ]);
    // The menu rooted at the channel scope proves the permission resolved.
    await screen.findByText("New group");
    await act(async () => {});
    expect(screen.getAllByText("New group")).toHaveLength(1);
  });
});
