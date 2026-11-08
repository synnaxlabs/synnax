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
  group,
  type resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  type BuiltInRole,
  createTestClient,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Group } from "@/platform/group";
import {
  createResource,
  createScopeTree,
  createState,
  findTreeRow,
  openTreeRootContextMenu,
  renderLiveTree,
} from "@/platform/tree/testutil";
import {
  awaitTextEditingElement,
  awaitTextEditingExit,
  createConsoleWrapper,
  renderSuspended,
  uniqueName,
} from "@/testutil";

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
        <Group.RootContextMenu
          key={root.type}
          root={root}
          state={createState([createResource(root, uniqueName("root"))])}
        />
      ))}
    </>,
    { wrapper },
  );
};

describe("Group.RootContextMenu", () => {
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

describe("Group.RootContextMenu in a tree", () => {
  const CHILD_COUNT = 150;
  let parent: group.Group;
  let names: string[];

  beforeAll(async () => {
    parent = await client.groups.create({
      scope: "channel",
      name: uniqueName("populated"),
    });
    names = Array.from({ length: CHILD_COUNT }, (_, i) =>
      // Zero padded so the tree's alphabetical sort matches creation order.
      uniqueName(`child-${String(i).padStart(3, "0")}`),
    );
    for (const name of names)
      await client.groups.create({ parent: group.resourceID(parent.key), name });
  });

  it("should start the rename of a new group outside the window", async () => {
    await renderLiveTree({
      client,
      root: channel.TYPE_RESOURCE_ID,
      contextMenus: Group.TREE_CONTEXT_MENUS,
      ...createScopeTree(parent.key),
    });
    await findTreeRow(names[CHILD_COUNT - 1]);
    expect(screen.queryByText(names[0])).toBeNull();
    await openTreeRootContextMenu();
    fireEvent.click(await screen.findByText("New group"));
    const editable = await awaitTextEditingElement();
    await act(async () => {
      fireEvent.keyDown(editable, { key: "Escape" });
    });
    await awaitTextEditingExit();
  });
});
