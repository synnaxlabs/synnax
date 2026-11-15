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
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { binary, uuid } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  type RenderOptions,
  screen,
  waitFor,
} from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { findButton } from "@/platform/modals/testutil";
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
import {
  assertDefined,
  awaitTextEditingElement,
  CaptureStatuses,
  commitTextEdit,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

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

// Clients cannot create internal channels, so this reads one the Core made.
const retrieveInternalChannel = async (): Promise<channelClient.Channel> => {
  const [ch] = await client.channels.retrieve({ internal: true, limit: 1 });
  assertDefined(ch);
  return ch;
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

    it("deletes a channel after confirmation", async () => {
      const ch = await createChannel();
      const host = hostChannels(ch);
      await renderChannelTree(host);
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${ch.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.channels.retrieve(ch.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });

    it("reports a delete that a task config refuses", async () => {
      const ch = await createChannel();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const device = await client.devices.create({
        key: uuid.create(),
        rack: rack.key,
        location: "127.0.0.1:502",
        name: uniqueName("plc"),
        make: "modbus",
        model: "modbus",
        properties: {},
      });
      const task = await rack.createTask({
        name: uniqueName("read"),
        type: "modbus_read",
        config: {
          device: device.key,
          channels: [{ type: "coil", key: "coil", channel: ch.key }],
        },
      });
      let statuses: Status.NotificationSpec[] = [];
      await renderChannelTree(
        hostChannels(ch),
        undefined,
        createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
      );
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${ch.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to delete channel",
            description:
              `cannot delete channel ${ch.name} because task ${task.name} uses it: ` +
              "validation error",
          }),
        ),
      );
      expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
    });

    it("opens the calculation editor for a calculated channel", async () => {
      const calc = await createChannel({
        isIndex: false,
        virtual: true,
        expression: "return 1",
      });
      const host = hostChannels(calc);
      // Monaco cannot run in jsdom, so the expression editor's boundary catches.
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
