// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import {
  act,
  fireEvent,
  type RenderOptions,
  screen,
  waitFor,
} from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import {
  findTreeRow,
  itemName,
  openTreeRowContextMenu,
  renderToolbar,
  searchTree,
} from "@/platform/tree/testutil";
import {
  assertDefined,
  awaitTextEditingElement,
  commitTextEdit,
  createTestClientWithGrants,
  getIconButton,
  getIconButtons,
  queryIcon,
  renderHookWithConsole,
  uniqueName,
  uniqueToken,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const renderChannelToolbar = async (
  onCaughtError?: RenderOptions["onCaughtError"],
): Promise<void> => {
  await renderToolbar(Channel.TOOLBAR.content, {
    client,
    items: Channel.TREE_ITEMS,
    onCaughtError,
  });
  await screen.findByText("Channels");
};

const createGroupedChannel = async (
  name: string,
): Promise<{ ch: channel.Channel; groupName: string }> => {
  const groupName = uniqueName("grp");
  const grp = await client.groups.create({ scope: "channel", name: groupName });
  const ch = await client.channels.create({
    name,
    dataType: DataType.TIMESTAMP,
    isIndex: true,
    group: grp.key,
  });
  return { ch, groupName };
};

describe("channel/Toolbar", () => {
  it("opens the create channel modal from the toolbar action", async () => {
    await renderChannelToolbar();
    // The calculated-channel action's composite icon also contains an add glyph, so
    // pick the add button that is not the calculated action.
    const create = await waitFor(() => {
      const calculated = getIconButton(document.body, "calculation");
      const btn = getIconButtons(document.body, "add").find((b) => b !== calculated);
      if (btn == null) throw new Error("create channel action not rendered");
      return btn;
    });
    fireEvent.click(create);
    expect(await screen.findByPlaceholderText("Channel name")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Index" })).toBeTruthy();
  });

  it("opens the create calculated channel modal from the toolbar action", async () => {
    // Monaco cannot run in jsdom, so the expression editor's boundary catches.
    await renderChannelToolbar(() => {});
    const button = await waitFor(() => getIconButton(document.body, "calculation"));
    await act(async () => fireEvent.click(button));
    expect(await screen.findByPlaceholderText("Channel name")).toBeTruthy();
    expect(await screen.findByText("Derivative")).toBeTruthy();
  });
});

describe("channel/Toolbar permissions", () => {
  it("should withhold the create actions from a viewer", async () => {
    await renderToolbar(Channel.TOOLBAR.content, {
      client: await roles.get("Viewer"),
      items: Channel.TREE_ITEMS,
    });
    await screen.findByText("Channels");
    // The tree can hold calculated channels, so only the header is checked.
    const header = document.querySelector(".console-toolbar__header");
    assertDefined(header);
    expect(getIconButtons(header, "add")).toHaveLength(0);
    expect(queryIcon(header, "calculation")).toBeNull();
  });

  it("should hide the toolbar itself from a subject who cannot read channels", async () => {
    const denied = await createTestClientWithGrants(client);
    assertDefined(Channel.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Channel.TOOLBAR.useVisible, {
      client: denied,
    });
    await waitFor(() => expect(result.current).toBe(false));
  });
});

describe("channel/Toolbar search", () => {
  it("should find a channel by name and show its group", async () => {
    const tok = uniqueToken();
    const name = `${tok}_flow`;
    const { groupName } = await createGroupedChannel(name);
    await renderChannelToolbar();
    searchTree(tok);
    const row = await findTreeRow(itemName(name));
    expect(row.querySelector("mark")?.textContent).toBe(tok);
    expect(row.textContent).toContain(groupName);
  });

  it("should rename a channel from a search result", async () => {
    const tok = uniqueToken();
    const name = `${tok}_flow`;
    const { ch } = await createGroupedChannel(name);
    await renderChannelToolbar();
    searchTree(tok);
    await openTreeRowContextMenu(itemName(name));
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditingElement();
    expect(editor.textContent).toBe(name);
    expect(editor.querySelector("mark")).toBeNull();
    const renamed = `${tok}_renamed`;
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.channels.retrieve(ch.key)).name).toBe(renamed),
    );
    expect(await findTreeRow(itemName(renamed))).toBeTruthy();
  });
});
