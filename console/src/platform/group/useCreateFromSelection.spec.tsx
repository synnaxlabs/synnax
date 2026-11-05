// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Group } from "@/platform/group";
import { type Tree } from "@/platform/tree";
import {
  createScopeTree,
  expandTreeRow,
  findTreeRow,
  getTreeRow,
  renderLiveTree,
  retrieveChildGroupKeys,
} from "@/platform/tree/testutil";
import {
  awaitTextEditingElement,
  awaitTextEditingExit,
  commitTextEdit,
  uniqueName,
  withControlHeld,
} from "@/testutil";

const client = createTestClient();

const GroupSelectionMenu = (props: Tree.ContextMenuProps): ReactElement => {
  const createFromSelection = Group.useCreateFromSelection();
  return <button onClick={() => createFromSelection(props)}>group selection</button>;
};

const createContainer = async () =>
  await client.groups.create({ scope: "channel", name: uniqueName("container") });

const renderScopeTree = async (container: group.Group) =>
  await renderLiveTree({
    client,
    root: channel.TYPE_RESOURCE_ID,
    ...createScopeTree(container.key, { ContextMenu: GroupSelectionMenu }),
  });

const groupSelection = async (first: string, last: string) => {
  fireEvent.click(await findTreeRow(first));
  withControlHeld(() => fireEvent.click(getTreeRow(last)));
  fireEvent.contextMenu(getTreeRow(last));
  fireEvent.click(await screen.findByText("group selection"));
  return await awaitTextEditingElement();
};

const setup = async () => {
  const container = await createContainer();
  const containerID = group.resourceID(container.key);
  const a = await client.groups.create({
    parent: containerID,
    name: uniqueName("child_a"),
  });
  const b = await client.groups.create({
    parent: containerID,
    name: uniqueName("child_b"),
  });
  await renderScopeTree(container);
  await screen.findByText(a.name);
  await screen.findByText(b.name);
  const editable = await groupSelection(a.name, b.name);
  return { containerID, a, b, editable };
};

const findGroup = async (name: string): Promise<group.Group> =>
  await waitFor(async () => {
    const [found] = await client.groups.retrieve({ searchTerm: name });
    if (found?.name !== name) throw new Error(`group ${name} not created`);
    return found;
  });

// The move reaches the client cache before the Core commits it, so a parent that reads
// as regrouped does not mean the new group holds its children yet.
const expectGrouped = async (key: group.Key, members: group.Group[]): Promise<void> =>
  await waitFor(async () =>
    expect(
      (await retrieveChildGroupKeys(client, group.resourceID(key))).sort(),
    ).toEqual(members.map((m) => m.key).sort()),
  );

describe("useCreateFromSelection", () => {
  it("should group a selection at the root under a new top-level group", async () => {
    const { containerID, a, b, editable } = await setup();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    const created = await findGroup(name);
    expect(created.scope).toBe("channel");
    expect(created.parent).toBeUndefined();
    await expectGrouped(created.key, [a, b]);
    expect(await retrieveChildGroupKeys(client, containerID)).toEqual([]);
  });

  it("should group a nested selection under its non-root parent", async () => {
    const container = await createContainer();
    const sub = await client.groups.create({
      parent: group.resourceID(container.key),
      name: uniqueName("sub"),
    });
    const subID = group.resourceID(sub.key);
    const a = await client.groups.create({ parent: subID, name: uniqueName("nest_a") });
    const b = await client.groups.create({ parent: subID, name: uniqueName("nest_b") });
    await renderScopeTree(container);
    await screen.findByText(sub.name);
    expandTreeRow(sub.name);
    await screen.findByText(a.name);
    await screen.findByText(b.name);
    const editable = await groupSelection(a.name, b.name);
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    const created = await findGroup(name);
    expect(created.parent).toEqual(subID);
    await expectGrouped(created.key, [a, b]);
  });

  it("should start the rename when the selection sits below the window", async () => {
    const container = await createContainer();
    const containerID = group.resourceID(container.key);
    const names = Array.from({ length: 150 }, (_, i) =>
      uniqueName(`child-${String(i).padStart(3, "0")}`),
    );
    for (const name of names) await client.groups.create({ parent: containerID, name });
    await renderScopeTree(container);
    await screen.findByText(names[0]);
    const [secondToLast, last] = names.slice(-2);
    const editable = await groupSelection(secondToLast, last);
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    const created = await findGroup(name);
    await waitFor(async () =>
      expect(await retrieveChildGroupKeys(client, containerID)).toHaveLength(
        names.length - 2,
      ),
    );
    expect(
      await retrieveChildGroupKeys(client, group.resourceID(created.key)),
    ).toHaveLength(2);
  });

  it("should not create a group and should restore the tree when the rename is empty", async () => {
    const { containerID, a, b, editable } = await setup();
    await act(async () => {
      commitTextEdit(editable, "");
    });
    await awaitTextEditingExit();
    await waitFor(() => {
      expect(screen.getByText(a.name)).toBeTruthy();
      expect(screen.getByText(b.name)).toBeTruthy();
    });
    expect((await retrieveChildGroupKeys(client, containerID)).sort()).toEqual(
      [a.key, b.key].sort(),
    );
  });

  it("should not create a group and should restore the tree when the rename is escaped", async () => {
    const { containerID, a, b, editable } = await setup();
    await act(async () => {
      fireEvent.keyDown(editable, { key: "Escape" });
    });
    await awaitTextEditingExit();
    await waitFor(() => {
      expect(screen.getByText(a.name)).toBeTruthy();
      expect(screen.getByText(b.name)).toBeTruthy();
    });
    expect((await retrieveChildGroupKeys(client, containerID)).sort()).toEqual(
      [a.key, b.key].sort(),
    );
  });
});
