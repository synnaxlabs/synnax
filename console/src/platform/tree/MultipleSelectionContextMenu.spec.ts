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
  DataType,
  group,
  label,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createScopeTree, getTreeRow, renderLiveTree } from "@/platform/tree/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  createTestClientWithGrants,
  uniqueName,
  withControlHeld,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

interface MixedSelection {
  child: group.Group;
  ch: channel.Channel;
}

const setupMixedSelection = async (as: Client = client): Promise<MixedSelection> => {
  const container = await client.groups.create({
    scope: "channel",
    name: uniqueName("container"),
  });
  const child = await client.groups.create({
    parent: group.resourceID(container.key),
    name: uniqueName("child"),
  });
  const ch = await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.FLOAT32,
    virtual: true,
    group: container.key,
  });
  await renderLiveTree({
    client: as,
    projectClient: client,
    root: channel.TYPE_RESOURCE_ID,
    ...createScopeTree(container.key),
  });
  await screen.findByText(child.name);
  await screen.findByText(ch.name);
  fireEvent.click(getTreeRow(child.name));
  withControlHeld(() => fireEvent.click(getTreeRow(ch.name)));
  fireEvent.contextMenu(getTreeRow(ch.name));
  return { child, ch };
};

describe("Tree.MultipleSelectionContextMenu", () => {
  it("should offer grouping and reload for a mixed-type selection", async () => {
    await setupMixedSelection();
    await screen.findByText("Group selection");
    await screen.findByText("Reload Console");
  });

  it("should group the mixed selection under a new group when the group item is clicked", async () => {
    const { child, ch } = await setupMixedSelection();
    fireEvent.click(await screen.findByText("Group selection"));
    const editable = await awaitTextEditingElement();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    const created = await waitFor(async () => {
      const [found] = await client.groups.retrieve({ searchTerm: name });
      if (found?.name !== name) throw new Error(`group ${name} not created`);
      return found;
    });
    expect(created.scope).toBe("channel");
    expect(created.parent).toBeUndefined();
    expect((await client.groups.retrieve({ key: child.key })).parent).toEqual(
      group.resourceID(created.key),
    );
    expect((await client.channels.retrieve(ch.key)).group).toBe(created.key);
  });
});

describe("Tree.MultipleSelectionContextMenu permissions", () => {
  it("should withhold grouping from a viewer", async () => {
    await setupMixedSelection(await roles.get("Viewer"));
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Group selection")).toBeNull();
  });

  it("should withhold grouping from a subject who cannot create groups", async () => {
    const writer = await createTestClientWithGrants(client, {
      retrieve: [
        group.TYPE_RESOURCE_ID,
        channel.TYPE_RESOURCE_ID,
        label.TYPE_RESOURCE_ID,
      ],
      update: [group.TYPE_RESOURCE_ID, channel.TYPE_RESOURCE_ID],
    });
    await setupMixedSelection(writer);
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Group selection")).toBeNull();
  });

  it("should offer grouping to an engineer", async () => {
    await setupMixedSelection(await roles.get("Engineer"));
    expect(await screen.findByText("Group selection")).toBeTruthy();
  });
});
