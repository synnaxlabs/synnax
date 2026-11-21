// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  failRequests,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  type RenderOptions,
  screen,
  waitFor,
} from "@testing-library/react";
import { createElement, Fragment } from "react";
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
  awaitTextEditingElement,
  CaptureStatuses,
  commitTextEdit,
  createTestClientWithGrants,
  getIconButton,
  getIconButtons,
  queryIcon,
  renderHookWithConsole,
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
    const create = await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(create);
    expect(await screen.findByPlaceholderText("Channel name")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Index" })).toBeTruthy();
  });

  it("opens the create calculated channel modal from the toolbar action", async () => {
    // Monaco cannot run in jsdom, so the expression editor's boundary catches.
    await renderChannelToolbar(() => {});
    const button = await waitFor(() =>
      getIconButton(document.body, "calculated-channel-create"),
    );
    await act(async () => fireEvent.click(button));
    expect(await screen.findByPlaceholderText("Channel name")).toBeTruthy();
    expect(await screen.findByText("Derivative")).toBeTruthy();
  });
});

describe("channel/Toolbar built-in keys", () => {
  it("should report a failed fetch of the built-in keys once", async () => {
    await client.channels.create(
      [0, 1].map(() => ({
        name: uniqueName("ch"),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      })),
    );
    // The client fetches the keys for itself too, and reports its own failure.
    const internal: Error[] = [];
    const failed = createTestClient({ onInternalError: (e) => internal.push(e) });
    failRequests(failed, "/channel/retrieve-built-ins", new Error("refused"));
    let statuses: Status.NotificationSpec[] = [];
    await renderToolbar(
      createElement(
        Fragment,
        null,
        Channel.TOOLBAR.content,
        createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
      ),
      { client: failed, items: Channel.TREE_ITEMS },
    );
    const failures = () =>
      statuses.filter(
        ({ message }) => message === "Failed to retrieve built-in channels",
      );
    await waitFor(() => expect(failures()).toHaveLength(1));
    // Each tree item reads the keys and fails again on its first mount.
    await waitFor(() =>
      expect(document.querySelectorAll(".lyra-tree__item").length).toBeGreaterThan(1),
    );
    await act(async () => {});
    expect(failures()).toHaveLength(1);
    expect(failures()[0].description).toBe(
      "Failed to retrieve built-in channels: refused",
    );
    await waitFor(() =>
      expect(internal.map((e) => e.message)).toEqual([
        "failed to fetch built-in channels",
      ]),
    );
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
    expect(queryIcon(header, "calculated-channel-create")).toBeNull();
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
