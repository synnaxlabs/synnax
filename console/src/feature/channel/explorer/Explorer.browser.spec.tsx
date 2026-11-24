// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  group,
  panel,
  type resource,
  type Synnax as Client,
  view,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { DataType } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Group } from "@/feature/group";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { createTabWrapper, findToolbarIconButton } from "@/platform/view/testutil";
import {
  createTestClientWithGrants,
  getBySelector,
  queryIconButton,
  renderLazy,
} from "@/testutil";

const client = createTestClient();

const TAB = { variant: "view", type: Channel.Explorer.TAB_TYPE } as const;

const { Content: Explorer } = Channel.Explorer.TAB;

interface Folders {
  outer: group.Group;
  inner: group.Group;
  floats: channel.Channel;
  ints: channel.Channel;
}

/** Creates a float and an int channel in a folder that sits in another folder. */
const createFolders = async (): Promise<Folders> => {
  const outer = await client.groups.create({
    scope: "channel",
    name: uniqueName("outer"),
  });
  const inner = await client.groups.create({
    parent: group.resourceID(outer.key),
    name: uniqueName("inner"),
  });
  const [floats, ints] = await client.channels.create([
    {
      name: uniqueName("floats"),
      dataType: DataType.FLOAT32,
      virtual: true,
      group: inner.key,
    },
    {
      name: uniqueName("ints"),
      dataType: DataType.INT64,
      virtual: true,
      group: inner.key,
    },
  ]);
  return { outer, inner, floats, ints };
};

// The Folders layout shows the resource tree, which reads its items from the provider.
const TREE_ITEMS: Tree.Items = {
  ...Channel.TREE_ITEMS,
  ...Group.createTreeItems([Channel.GROUP_MEMBER]),
};

const TreeExplorer = (): ReactElement => (
  <Haul.Provider>
    <Tree.Provider items={TREE_ITEMS}>
      <Explorer />
      <Modals.Stack />
    </Tree.Provider>
  </Haul.Provider>
);

const findBySelector = async (selector: string): Promise<HTMLElement> =>
  await waitFor(() => getBySelector(document.body, selector));

const renderView = async (stored: view.View): Promise<void> => {
  const { wrapper } = await createTabWrapper({
    client,
    tab: { variant: "resource", resource: view.resourceID(stored.key) },
  });
  await renderLazy(<TreeExplorer />, { wrapper });
};

describe("channel explorer", () => {
  it("should show the matches of a filter under their folders", async () => {
    const { outer, inner, floats, ints } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [inner.key], dataType: { in: [DataType.FLOAT32] } },
      }),
    );
    const tree = within(await findBySelector(".console-channel-explorer__folders"));
    expect(await tree.findByText(floats.name)).toBeTruthy();
    expect(tree.getByText(inner.name)).toBeTruthy();
    expect(tree.getByText(outer.name)).toBeTruthy();
    expect(tree.queryByText(ints.name)).toBeNull();
  });

  it("should show a chip for each field of a stored filter", async () => {
    const { inner } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: {
          groups: [inner.key],
          dataType: { in: [DataType.FLOAT32] },
          virtual: true,
        },
      }),
    );
    const chips = within(await findBySelector(".console-view__chips"));
    expect(await chips.findByText(inner.name)).toBeTruthy();
    expect(chips.getByText("Folder")).toBeTruthy();
    expect(chips.getByText("Data type")).toBeTruthy();
    expect(chips.getByText("float32")).toBeTruthy();
    expect(chips.getByText("Kind")).toBeTruthy();
    expect(chips.getByText("Virtual")).toBeTruthy();
  });

  it("should list each channel with the folders that hold it", async () => {
    const { outer, inner, floats, ints } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [inner.key] },
        layout: "list",
      }),
    );
    expect(await screen.findByText(floats.name)).toBeTruthy();
    expect(screen.getByText(ints.name)).toBeTruthy();
    await waitFor(() =>
      expect(screen.getAllByText(`${outer.name} / ${inner.name}`)).toHaveLength(2),
    );
    expect(screen.getByText("int64")).toBeTruthy();
  });

  it("should offer to clear a filter that matches nothing, in each layout", async () => {
    const { outer } = await createFolders();
    for (const layout of view.CHANNEL_LAYOUTS) {
      await renderView(
        await client.views.create({
          name: uniqueName("view"),
          type: "channel",
          where: { groups: [outer.key] },
          layout,
        }),
      );
      expect(await screen.findByText("No channels match the filter")).toBeTruthy();
      fireEvent.click(screen.getByText("Clear filters"));
      await waitFor(() => expect(screen.queryByText("Folder")).toBeNull());
      cleanup();
    }
  });

  it("should offer the actions on a channel in the List layout", async () => {
    const { inner, floats } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [inner.key] },
        layout: "list",
      }),
    );
    fireEvent.contextMenu(await screen.findByText(floats.name));
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Copy link")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
  });

  it("should save the layout of a stored view", async () => {
    const stored = await client.views.create({
      name: uniqueName("view"),
      type: "channel",
    });
    await renderView(stored);
    fireEvent.click(await screen.findByRole("button", { name: "List" }));
    await waitFor(async () => {
      const saved = await client.views.retrieve(stored.key);
      expect(saved.type === "channel" && saved.layout).toBe("list");
    });
  });
});

describe("channel explorer permissions", () => {
  const createEditor = async (creatable: resource.ID[] = []) =>
    await createTestClientWithGrants(client, {
      retrieve: [
        channel.TYPE_RESOURCE_ID,
        view.TYPE_RESOURCE_ID,
        group.TYPE_RESOURCE_ID,
        panel.TYPE_RESOURCE_ID,
      ],
      update: [view.TYPE_RESOURCE_ID],
      create: [view.TYPE_RESOURCE_ID, ...creatable],
    });

  const renderExplorer = async (as: Client) => {
    const { wrapper } = await createTabWrapper({ client, as, tab: TAB });
    await renderLazy(<TreeExplorer />, { wrapper });
  };

  it("should withhold the create buttons from a subject who cannot create channels", async () => {
    await renderExplorer(await createEditor());
    await waitFor(() =>
      expect(
        queryIconButton(getBySelector(document.body, ".console-view__views"), "add"),
      ).toBeTruthy(),
    );
    expect(
      queryIconButton(getBySelector(document.body, ".console-view__toolbar"), "add"),
    ).toBeNull();
  });

  it("should offer the create button to a subject who may create channels", async () => {
    await renderExplorer(await createEditor([channel.TYPE_RESOURCE_ID]));
    expect(await findToolbarIconButton("add")).toBeTruthy();
  });
});
