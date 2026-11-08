// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, group, resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Channel } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";
import {
  createGroupItem,
  createSearchTree,
  findTreeRow,
  itemName,
  renderLiveTree,
  retrieveChildGroupKeys,
  searchTree,
} from "@/platform/tree/testutil";
import { getIconButton, uniqueName, uniqueToken, withControlHeld } from "@/testutil";

const client = createTestClient();

const GROUP_ITEMS: Tree.Items = { group: createGroupItem() };

const useGroupSearch = Tree.createSearch({
  noun: "groups",
  useMatches: Group.useMatches,
  usePlacement: Group.usePlacement,
});

const renderSearchable = async (
  root: resource.ID,
  search: Tree.UseSearch = useGroupSearch,
) => await renderLiveTree({ client, root, items: GROUP_ITEMS, search });

const hiddenTrees = (): number =>
  document.querySelectorAll(".lyra-tree.console-tree--hidden").length;

const childKeysOf = async (id: resource.ID): Promise<string[]> =>
  await retrieveChildGroupKeys(client, id);

describe("Group search", () => {
  it("should show each match with its group and hide the tree", async () => {
    const { root, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    await findTreeRow(names.echo);
    searchTree(tok);
    const charlie = await findTreeRow(itemName(names.charlie));
    expect(charlie.textContent).toContain(names.bravo);
    expect(charlie.querySelector("mark")?.textContent).toBe(tok);
    await findTreeRow(itemName(names.alpha));
    await findTreeRow(itemName(names.delta));
    expect(hiddenTrees()).toBe(1);
    const results = document.querySelector(".lyra-tree:not(.console-tree--hidden)");
    expect(results?.textContent).not.toContain(names.echo);
  });

  it("should find the matches of a custom source with the groups", async () => {
    const { root, bravo, tok, names } = await createSearchTree(client);
    const ch = await client.channels.create({
      name: `${tok}_foxtrot`,
      dataType: DataType.FLOAT32,
      virtual: true,
      group: bravo,
    });
    const useMatches: Tree.UseMatches = (query) => {
      const channels = Channel.useResultList(
        query == null ? null : { searchTerm: query.term, limit: query.limit },
      );
      return [
        ...Group.useMatches(query),
        Tree.toMatches(channels, (c) => Tree.toResource("channel")(c.payload)),
      ];
    };
    await renderSearchable(
      root,
      Tree.createSearch({
        noun: "things",
        useMatches,
        usePlacement: Group.usePlacement,
      }),
    );
    searchTree(tok);
    const row = await findTreeRow(itemName(ch.name));
    expect(row.textContent).toContain(names.bravo);
    await findTreeRow(itemName(names.charlie));
  });

  it("should show the matches inside their groups", async () => {
    const { root, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    searchTree(tok);
    await findTreeRow(itemName(names.charlie));
    fireEvent.click(getIconButton(document.body, "tree-view"));
    const bravo = await findTreeRow(names.bravo);
    const charlie = await findTreeRow(itemName(names.charlie));
    expect(Number(charlie.getAttribute("aria-level"))).toBe(
      Number(bravo.getAttribute("aria-level")) + 1,
    );
    expect(charlie.textContent).not.toContain(names.bravo);
  });

  it("should show every child of a matched group, and only the matches of the rest", async () => {
    const { root, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    searchTree(tok);
    await findTreeRow(itemName(names.charlie));
    fireEvent.click(getIconButton(document.body, "tree-view"));
    expect(await findTreeRow(names.golf)).toBeTruthy();
    expect(
      screen.queryByText(names.hotel, { ignore: ".console-tree--hidden *" }),
    ).toBeNull();
  });

  it("should drop the other children of a group that stops matching", async () => {
    const { root, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    searchTree(tok);
    await findTreeRow(itemName(names.charlie));
    fireEvent.click(getIconButton(document.body, "tree-view"));
    await findTreeRow(names.golf);
    fireEvent.change(screen.getByPlaceholderText(/^Search /), {
      target: { value: names.charlie },
    });
    await waitFor(() =>
      expect(
        screen.queryByText(names.golf, { ignore: ".console-tree--hidden *" }),
      ).toBeNull(),
    );
    expect(await findTreeRow(itemName(names.charlie))).toBeTruthy();
  });

  it("should show every child of an open group that starts to match", async () => {
    const { root, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    searchTree(names.charlie);
    await findTreeRow(itemName(names.charlie));
    fireEvent.click(getIconButton(document.body, "tree-view"));
    await findTreeRow(names.bravo);
    expect(
      screen.queryByText(names.golf, { ignore: ".console-tree--hidden *" }),
    ).toBeNull();
    fireEvent.change(screen.getByPlaceholderText(/^Search /), {
      target: { value: tok },
    });
    expect(await findTreeRow(names.golf)).toBeTruthy();
  });

  it("should show every child of a matched group that matches again", async () => {
    const { root, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    searchTree(tok);
    await findTreeRow(itemName(names.charlie));
    fireEvent.click(getIconButton(document.body, "tree-view"));
    await findTreeRow(names.golf);
    const field = screen.getByPlaceholderText(/^Search /);
    fireEvent.change(field, { target: { value: names.echo } });
    await waitFor(() =>
      expect(
        screen.queryByText(names.golf, { ignore: ".console-tree--hidden *" }),
      ).toBeNull(),
    );
    fireEvent.change(field, { target: { value: tok } });
    expect(await findTreeRow(names.golf)).toBeTruthy();
  });

  it("should show the new name of a group above a match", async () => {
    const { root, bravo, tok, names } = await createSearchTree(client);
    await renderSearchable(root);
    searchTree(tok);
    await findTreeRow(itemName(names.charlie));
    const renamed = uniqueName("bravo");
    await client.groups.rename(bravo, renamed);
    await waitFor(async () =>
      expect((await findTreeRow(itemName(names.charlie))).textContent).toContain(
        renamed,
      ),
    );
  });

  it("should drop a match when a group above it moves out of the root", async () => {
    const { root, bravo, tok, names } = await createSearchTree(client);
    const other = await client.groups.create({
      scope: "channel",
      name: uniqueName("other"),
    });
    await renderSearchable(root);
    searchTree(tok);
    await findTreeRow(itemName(names.charlie));
    await client.groups.setParent(bravo, group.resourceID(other.key));
    await waitFor(() =>
      expect(
        screen.queryByText(itemName(names.charlie), {
          ignore: ".console-tree--hidden *",
        }),
      ).toBeNull(),
    );
    expect(await findTreeRow(itemName(names.delta))).toBeTruthy();
  });
});

describe("Group search selection", () => {
  it("should give no parent when the selection sits under more than one parent", async () => {
    const tok = uniqueToken();
    const root = await client.groups.create({
      scope: "channel",
      name: uniqueName("root"),
    });
    const rootID = group.resourceID(root.key);
    const left = await client.groups.create({
      parent: rootID,
      name: uniqueName("left"),
    });
    const right = await client.groups.create({
      parent: rootID,
      name: uniqueName("right"),
    });
    const a = await client.groups.create({
      parent: group.resourceID(left.key),
      name: `${tok} a`,
    });
    const b = await client.groups.create({
      parent: group.resourceID(right.key),
      name: `${tok} b`,
    });
    const menuSpy = vi.fn();
    const TreeContextMenu = (props: Tree.ContextMenuProps): ReactElement => {
      menuSpy(props);
      return <span>group service menu</span>;
    };
    const items: Tree.Items = {
      group: createGroupItem({ ContextMenu: TreeContextMenu }),
    };
    await renderLiveTree({ client, root: rootID, items, search: useGroupSearch });
    searchTree(tok);
    fireEvent.click(await findTreeRow(itemName(a.name)));
    const rowB = await findTreeRow(itemName(b.name));
    withControlHeld(() => fireEvent.click(rowB));
    fireEvent.contextMenu(rowB);
    await screen.findByText("group service menu");
    const [props] = menuSpy.mock.lastCall as [Tree.ContextMenuProps];
    expect(props.selection.ids).toHaveLength(2);
    expect(props.selection.parentID).toBeNull();
  });

  it("should move only the group when a group and a match inside it are dragged from search results", async () => {
    const tok = uniqueToken();
    const container = await client.groups.create({
      scope: "channel",
      name: uniqueName("container"),
    });
    const containerID = group.resourceID(container.key);
    const folder = await client.groups.create({
      parent: containerID,
      name: `${tok} folder`,
    });
    const folderID = group.resourceID(folder.key);
    const report = await client.groups.create({
      parent: folderID,
      name: `${tok} report`,
    });
    const dest = await client.groups.create({
      parent: containerID,
      name: `${tok} dest`,
    });
    await renderLiveTree({
      client,
      root: containerID,
      items: { group: createGroupItem({ canDrop: () => true }) },
      search: useGroupSearch,
    });
    searchTree(tok);
    // The report is clicked first: a click on the folder opens it and shows the
    // report a second time.
    fireEvent.click(await findTreeRow(itemName(report.name)));
    // The report shows the folder's name as its group, so the ID finds the folder.
    const folderRow = document.querySelector(
      `.lyra-tree:not(.console-tree--hidden) [id="${resource.idToString(folderID)}"]`,
    );
    if (folderRow == null) throw new Error("folder result not found");
    withControlHeld(() => fireEvent.click(folderRow));
    fireEvent.dragStart(folderRow);
    fireEvent.drop(await findTreeRow(itemName(dest.name)));
    await waitFor(async () =>
      expect(await childKeysOf(group.resourceID(dest.key))).toContain(folder.key),
    );
    expect(await childKeysOf(group.resourceID(dest.key))).not.toContain(report.key);
    expect(await childKeysOf(folderID)).toContain(report.key);
  });
});
