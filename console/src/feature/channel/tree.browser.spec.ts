// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel as channelClient,
  DataType,
  lineplot,
  NotFoundError,
  query,
} from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestClientWithPolicy,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { binary, uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  render,
  type RenderOptions,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Channel } from "@/feature/channel";
import { createReadTask } from "@/feature/channel/testutil";
import { findLastButton, findSurface } from "@/platform/modals/testutil";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  findTreeRow,
  HOST_ID,
  openTreeRowContextMenu,
  renderLiveTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { awaitTextEditingElement, commitTextEdit, resolveFocusedTab } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Channel.TREE_ITEMS.channel;

const createChannel = async (overrides: Partial<channelClient.New> = {}) =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
    ...overrides,
  });

/** Waits for the dialog's Delete button to finish planning, then clicks it. */
const clickDelete = async () => {
  await waitFor(() =>
    expect(findLastButton("Delete").getAttribute("aria-disabled")).toBeNull(),
  );
  fireEvent.click(findLastButton("Delete"));
};

const expectGone = async (retrieve: () => Promise<unknown>) =>
  await waitFor(async () => {
    await expect(retrieve()).rejects.toSatisfy((e) => NotFoundError.matches(e));
  });

// Clients cannot create internal channels, so this reads one the Core made.
const retrieveInternalChannel = async (): Promise<channelClient.Channel> => {
  const [ch] = await client.channels.retrieve({ internal: true, limit: 1 });
  assertDefined(ch);
  return ch;
};

/**
 * Creates a user whose role may retrieve and delete ch, and returns its client. A spec
 * deletes the retrieve policy to make the next plan fail, and grants it again to retry.
 */
const createChannelUser = async (ch: channelClient.Channel) => {
  const role = await createEmptyRole(client);
  const grant = async (action: "retrieve" | "delete") =>
    await client.access.policies.create({
      name: uniqueName("impact"),
      role: role.key,
      objects: [channelClient.resourceID(ch.key)],
      actions: [action],
    });
  await grant("delete");
  const retrieve = await grant("retrieve");
  const username = uniqueName("user");
  await client.users.create({
    username,
    password: "test",
    name: "test",
    role: role.key,
  });
  return {
    userClient: createTestClientAs(username, "test"),
    retrieve,
    grantRetrieve: async () => await grant("retrieve"),
  };
};

// Hosts the channels under the host root, outside any group.
const hostChannels = (...channels: { key: channelClient.Key }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.channels,
      query.keysRequest(channels.map((ch) => ch.key)),
      (ch) => Tree.toResource("channel")(ch.payload),
    ),
  ]);

const renderChannelTree = async (
  host: ReturnType<typeof hostChannels>,
  onCaughtError?: RenderOptions["onCaughtError"],
  extra?: ReactNode,
) =>
  await renderLiveTree({
    client,
    root: HOST_ID,
    items: { ...Channel.TREE_ITEMS, ...host },
    onCaughtError,
    extra,
  });

describe("channel/tree", () => {
  describe("haulItems", () => {
    it("hauls a regular channel as a channel item", async () => {
      const ch = await createChannel();
      const items = Item.haulItems(
        createResource(channelClient.resourceID(ch.key), ch.name),
        client,
      );
      expect(items).toHaveLength(1);
      expect(items[0].type).toBe("channel");
      expect(items[0].key).toBe(ch.key);
    });

    it("hauls an internal channel as a schematic value element only", async () => {
      const ch = await retrieveInternalChannel();
      const items = Item.haulItems(
        createResource(channelClient.resourceID(ch.key), ch.name),
        client,
      );
      expect(items).toHaveLength(1);
      expect(items[0].type).toBe("schematic-element");
    });
  });

  describe("haulPreview", () => {
    // The preview renders above every provider, so the spec renders it with none.
    const factsOf = (ch: { key: channelClient.Key; name: string }): string | null => {
      const resource = createResource(channelClient.resourceID(ch.key), ch.name);
      const preview = Item.haulPreview({
        resource,
        client,
        count: 1,
        resolveColor: () => undefined,
      });
      const { container } = render(preview);
      return container.querySelector(".lyra-haul-preview__facts")?.textContent ?? null;
    };

    it("previews a channel with its data type and index", async () => {
      const index = await createChannel();
      const ch = await createChannel({
        dataType: DataType.FLOAT64,
        isIndex: false,
        index: index.key,
      });
      expect(factsOf(ch)).toBe(`float64·${index.name}`);
    });

    it("previews a channel missing from the cache by its type name", () => {
      expect(factsOf({ key: 2 ** 20, name: "uncached" })).toBe("Channel");
    });
  });

  describe("onSelect", () => {
    it("creates a line plot for the selection when no plot is focused", async () => {
      const ch = await createChannel();
      const proj = await client.projects.create({
        name: uniqueName("proj"),
      });
      const host = hostChannels(ch);
      const { store } = await renderChannelTree(host);
      store.dispatch(Session.Project.select(proj.key));
      fireEvent.doubleClick(await findTreeRow(ch.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.type).toBe("lineplot");
      const plot = await client.lineplots.retrieve(tab.resource.key);
      expect(plot.name).toBe("Line plot");
      expect(plot.channels.y1).toContain(ch.key);
    });

    it("adds the selection to the focused line plot", async () => {
      const ch = await createChannel();
      const proj = await client.projects.create({
        name: uniqueName("proj"),
      });
      const plot = await client.lineplots.create({
        project: proj.key,
        name: uniqueName("plot"),
      });
      const host = hostChannels(ch);
      const { store } = await renderChannelTree(host);
      const tabKey = uuid.create();
      const pan = await client.panels.create({
        name: uniqueName("panel"),
        project: proj.key,
        root: {
          variant: "leaf",
          tabs: [
            {
              variant: "resource",
              key: tabKey,
              resource: lineplot.resourceID(plot.key),
            },
          ],
        },
      });
      store.dispatch(Session.Project.select(proj.key));
      store.dispatch(Session.Panel.select({ key: pan.key }));
      store.dispatch(
        Session.Panel.internalSelectTab({
          key: pan.key,
          tabKey,
          otherTabKeys: [tabKey],
        }),
      );
      fireEvent.doubleClick(await findTreeRow(ch.name));
      await waitFor(async () => {
        const { channels } = await client.lineplots.retrieve(plot.key);
        expect(channels.y1).toContain(ch.key);
      });
    });

    it("opens a virtual channel without an expression in a log", async () => {
      const virtualCh = await createChannel({ isIndex: false, virtual: true });
      const proj = await client.projects.create({
        name: uniqueName("proj"),
      });
      const host = hostChannels(virtualCh);
      const { store } = await renderChannelTree(host);
      store.dispatch(Session.Project.select(proj.key));
      fireEvent.doubleClick(await findTreeRow(virtualCh.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.type).toBe("log");
      const opened = await client.logs.retrieve(tab.resource.key);
      expect(opened.name).toBe("Log");
      expect(opened.channels.map((e) => e.channel)).toContain(virtualCh.key);
    });
  });

  describe("tree", () => {
    it("renders channels through the custom item", async () => {
      const ch = await createChannel();
      const host = hostChannels(ch);
      await renderChannelTree(host);
      expect(await screen.findByText(ch.name)).toBeTruthy();
    });

    it("renames a channel from the context menu", async () => {
      const ch = await createChannel();
      const host = hostChannels(ch);
      await renderChannelTree(host);
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditingElement();
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.channels.retrieve(ch.key)).name).toBe(renamed),
      );
    });

    it("deletes a channel that nothing else uses", async () => {
      const ch = await createChannel();
      await renderChannelTree(hostChannels(ch));
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Delete channel ${ch.name}`);
      await screen.findByText(
        "Deletes the channel and its data. Nothing else uses it. " +
          "This action cannot be undone.",
      );
      await clickDelete();
      await expectGone(async () => await client.channels.retrieve(ch.key));
    });

    it("deletes a task that blocks the delete once the user picks it", async () => {
      const ch = await createChannel();
      const task = await createReadTask(client, [ch.key]);
      await renderChannelTree(hostChannels(ch));
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText("Blocks the delete");
      await screen.findByText(task.name);
      await clickDelete();
      await screen.findByText("Uses the channel");
      await screen.findByText("1 record blocks the delete");
      screen.getByText('Check "Delete too", or cancel.');
      expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
      fireEvent.click(
        screen.getByRole("checkbox", { name: `${task.name} Delete too` }),
      );
      await waitFor(() => expect(screen.queryByText("Uses the channel")).toBeNull());
      await clickDelete();
      await expectGone(async () => await client.channels.retrieve(ch.key));
      await expectGone(async () => await client.tasks.retrieve({ key: task.key }));
    });

    it("keeps the dialog unseen until its first plan arrives", async () => {
      const ch = await createChannel();
      await createReadTask(client, [ch.key]);
      await renderChannelTree(hostChannels(ch));
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      // The opacity reads the reveal: vitest's toBeVisible counts an opacity of 0 as
      // visible, as Playwright does.
      const surface = findSurface();
      expect(getComputedStyle(surface).opacity).toBe("0");
      await screen.findByRole("region", { name: "Blocks the delete" });
      await waitFor(() => expect(getComputedStyle(surface).opacity).toBe("1"));
      expect(screen.getByRole("dialog")).toHaveFocus();
    });

    it("shows the spinner when the plan is slow, then the records", async () => {
      const ch = await createChannel();
      await createReadTask(client, [ch.key]);
      await renderChannelTree(hostChannels(ch));
      await openTreeRowContextMenu(ch.name);
      const item = await screen.findByText("Delete");
      // Everything under fake timers runs at once, so the plan is still in flight when
      // the spinner delay ends.
      vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
      let surface: HTMLElement;
      try {
        fireEvent.click(item);
        surface = findSurface();
        expect(getComputedStyle(surface).opacity).toBe("0");
        act(() => {
          vi.advanceTimersByTime(1000);
        });
      } finally {
        vi.useRealTimers();
      }
      screen.getByText("Listing the records that this delete touches");
      await waitFor(() => expect(getComputedStyle(surface).opacity).toBe("1"));
      await screen.findByRole("region", { name: "Blocks the delete" });
      expect(
        screen.queryByText("Listing the records that this delete touches"),
      ).toBeNull();
    });

    it("deletes every task that blocks the delete with one check", async () => {
      const ch = await createChannel();
      const first = await createReadTask(client, [ch.key]);
      const second = await createReadTask(client, [ch.key]);
      await renderChannelTree(hostChannels(ch));
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(first.name);
      await screen.findByText(second.name);
      fireEvent.click(await screen.findByRole("checkbox", { name: "Delete all too" }));
      await waitFor(() =>
        expect(
          screen.getByRole("checkbox", { name: `${second.name} Delete too` }),
        ).toBeChecked(),
      );
      expect(
        screen.getByRole("checkbox", { name: `${first.name} Delete too` }),
      ).toBeChecked();
      await clickDelete();
      await expectGone(async () => await client.channels.retrieve(ch.key));
      await expectGone(async () => await client.tasks.retrieve({ key: first.key }));
      await expectGone(async () => await client.tasks.retrieve({ key: second.key }));
    });

    it("shows the new plan when a holder appears before the delete", async () => {
      const ch = await createChannel({
        dataType: DataType.FLOAT32,
        isIndex: false,
        virtual: true,
      });
      await renderChannelTree(hostChannels(ch));
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(
        "Deletes the channel and its data. Nothing else uses it. " +
          "This action cannot be undone.",
      );
      const proj = await client.projects.create({ name: uniqueName("proj") });
      const plot = await client.lineplots.create({
        project: proj.key,
        name: uniqueName("plot"),
        channels: { y1: [ch.key] },
      });
      await clickDelete();
      await screen.findByText("These records changed. Check them again.");
      await screen.findByText("Loses the channel");
      await screen.findByText(plot.name);
      expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
    });

    it("tells the user to pick a hidden task that blocks the delete", async () => {
      const ch = await createChannel();
      await createReadTask(client, [ch.key]);
      const userClient = await createTestClientWithPolicy(client, {
        name: uniqueName("impact"),
        objects: [channelClient.resourceID(ch.key)],
        actions: ["retrieve", "delete"],
      });
      assertDefined(Item.ContextMenu);
      await renderTreeContextMenu(Item.ContextMenu, {
        client: userClient,
        resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
      });
      fireEvent.click(await screen.findByText("Delete"));
      const blocks = await screen.findByRole("region", { name: "Blocks the delete" });
      const hidden = within(blocks).getByRole("checkbox", {
        name: "1 record you cannot see Delete too",
      });
      await clickDelete();
      const blocked = "1 record blocks the delete";
      await screen.findByText(blocked);
      screen.getByText('Check "Delete too", or cancel.');
      expect(hidden).toHaveAccessibleDescription("Uses the channel");
      expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
      fireEvent.click(hidden);
      await waitFor(() => expect(screen.queryByText(blocked)).toBeNull());
      expect(screen.queryByText("Uses the channel")).toBeNull();
    });

    it("lists a hidden plot with no check when nothing blocks", async () => {
      const ch = await createChannel();
      const proj = await client.projects.create({ name: uniqueName("proj") });
      await client.lineplots.create({
        project: proj.key,
        name: uniqueName("plot"),
        channels: { y1: [ch.key] },
      });
      const userClient = await createTestClientWithPolicy(client, {
        name: uniqueName("impact"),
        objects: [channelClient.resourceID(ch.key)],
        actions: ["retrieve", "delete"],
      });
      assertDefined(Item.ContextMenu);
      await renderTreeContextMenu(Item.ContextMenu, {
        client: userClient,
        resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
      });
      fireEvent.click(await screen.findByText("Delete"));
      const loses = await screen.findByRole("region", { name: "Loses the channel" });
      within(loses).getByText("1 record you cannot see");
      expect(within(loses).queryByRole("checkbox")).toBeNull();
      expect(screen.queryByRole("region", { name: "Blocks the delete" })).toBeNull();
    });

    it("plans again when the user presses Try again", async () => {
      const ch = await createChannel();
      const { userClient, retrieve, grantRetrieve } = await createChannelUser(ch);
      assertDefined(Item.ContextMenu);
      await renderTreeContextMenu(Item.ContextMenu, {
        client: userClient,
        resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
      });
      const item = await screen.findByText("Delete");
      // The plan needs retrieve on the channel, so losing it fails the plan.
      await client.access.policies.delete(retrieve.key);
      fireEvent.click(item);
      const failed = "Failed to list the records that this delete touches";
      await screen.findByText(failed);
      expect(findLastButton("Delete")).toHaveAttribute("aria-disabled", "true");
      await grantRetrieve();
      const retry = findLastButton("Try again");
      retry.focus();
      fireEvent.click(retry);
      expect(screen.getByRole("dialog")).toHaveFocus();
      await screen.findByText(
        "Deletes the channel and its data. Nothing else uses it. " +
          "This action cannot be undone.",
      );
      expect(screen.queryByText(failed)).toBeNull();
      await clickDelete();
      await expectGone(async () => await client.channels.retrieve(ch.key));
    });

    it("keeps the records listed when a plan after a pick fails", async () => {
      const ch = await createChannel();
      await createReadTask(client, [ch.key]);
      const { userClient, retrieve, grantRetrieve } = await createChannelUser(ch);
      assertDefined(Item.ContextMenu);
      await renderTreeContextMenu(Item.ContextMenu, {
        client: userClient,
        resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
      });
      fireEvent.click(await screen.findByText("Delete"));
      const hidden = await screen.findByRole("checkbox", {
        name: "1 record you cannot see Delete too",
      });
      // The plan needs retrieve on the channel, so losing it fails the next plan.
      await client.access.policies.delete(retrieve.key);
      hidden.focus();
      fireEvent.click(hidden);
      const failed = "Failed to update the records";
      await screen.findByText(failed);
      expect(hidden).toHaveFocus();
      expect(findLastButton("Delete")).toHaveAttribute("aria-disabled", "true");
      await grantRetrieve();
      fireEvent.click(findLastButton("Try again"));
      await waitFor(() => expect(screen.queryByText(failed)).toBeNull());
      await waitFor(() =>
        expect(findLastButton("Delete").getAttribute("aria-disabled")).toBeNull(),
      );
      expect(hidden).toBeChecked();
    });

    it("opens the calculation editor for a calculated channel", async () => {
      const calc = await createChannel({
        isIndex: false,
        virtual: true,
        expression: "return 1",
      });
      const host = hostChannels(calc);
      await renderChannelTree(host, () => {});
      await openTreeRowContextMenu(calc.name);
      const edit = await screen.findByText("Edit calculation");
      await act(async () => fireEvent.click(edit));
      expect(await screen.findByDisplayValue(calc.name)).toBeTruthy();
    });

    it("should offer Reload Console", async () => {
      const ch = await createChannel();
      assertDefined(Item.ContextMenu);
      await renderTreeContextMenu(Item.ContextMenu, {
        client,
        resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
      });
      expect(await screen.findByText("Reload Console")).toBeTruthy();
    });

    it("withholds rename from an internal channel", async () => {
      const ch = await retrieveInternalChannel();
      assertDefined(Item.ContextMenu);
      await renderTreeContextMenu(Item.ContextMenu, {
        client,
        resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
      });
      // Delete shares the update permission the rename item needs, so its presence
      // proves the gate resolved before the absence below is read.
      expect(await screen.findByText("Delete")).toBeTruthy();
      expect(screen.queryByText("Rename")).toBeNull();
    });
  });
});

describe("copy properties", () => {
  it("should copy the stored channel as JSON", async () => {
    const writeText = stubClipboardWriteText();
    const ch = await createChannel();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
    });
    fireEvent.click(await screen.findByText("Copy properties"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(
        binary.JSON_CODEC.encodeString(ch.payload),
      ),
    );
  });
});

describe("permission to write the channel", () => {
  it("should withhold rename, grouping, and delete from a viewer", async () => {
    const ch = await createChannel();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [createResource(channelClient.resourceID(ch.key), ch.name)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Group selection")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});
