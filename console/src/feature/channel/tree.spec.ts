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
  group,
  lineplot,
  NotFoundError,
  ontology,
  project,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { findButton } from "@/platform/modals/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import {
  findTreeRow,
  openTreeRowContextMenu,
  renderOntologyTree,
} from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditingElement,
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

const createChannelGroup = async (
  ...channels: { key: channelClient.Key }[]
): Promise<ontology.ID> => {
  const grp = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("chgrp"),
  });
  const channelsGroup = await client.channels.retrieveGroup();
  await client.ontology.moveChildren(
    group.ontologyID(channelsGroup.key),
    group.ontologyID(grp.key),
    ...channels.map((c) => channelClient.ontologyID(c.key)),
  );
  return group.ontologyID(grp.key);
};

const renderChannelTree = async (root: ontology.ID) =>
  await renderOntologyTree({
    client,
    root,
    items: Channel.TREE_ITEMS,
  });

describe("channel/ontology", () => {
  describe("haulItems", () => {
    it("hauls a regular channel as a channel item", async () => {
      const ch = await createChannel();
      const items = Item.haulItems(
        createResource(channelClient.ontologyID(ch.key), ch.name, {
          internal: false,
        }),
      );
      expect(items).toHaveLength(1);
      expect(items[0].type).toBe("channel");
      expect(items[0].key).toBe(ch.key);
    });

    it("hauls an internal channel as a schematic value element only", async () => {
      const ch = await createChannel();
      const items = Item.haulItems(
        createResource(channelClient.ontologyID(ch.key), ch.name, {
          internal: true,
        }),
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
        layout: {},
      });
      const root = await createChannelGroup(ch);
      const { store } = await renderChannelTree(root);
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
        layout: {},
      });
      const plot = await client.lineplots.create(proj.key, {
        name: uniqueName("plot"),
      });
      const root = await createChannelGroup(ch);
      const { store } = await renderChannelTree(root);
      const tabKey = uuid.create();
      const pan = await client.panels.create({
        name: uniqueName("panel"),
        parent: project.ontologyID(proj.key),
        root: {
          variant: "leaf",
          tabs: [
            {
              variant: "resource",
              key: tabKey,
              resource: lineplot.ontologyID(plot.key),
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
        layout: {},
      });
      const root = await createChannelGroup(virtualCh);
      const { store } = await renderChannelTree(root);
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
      const root = await createChannelGroup(ch);
      await renderChannelTree(root);
      expect(await screen.findByText(ch.name)).toBeTruthy();
    });

    it("renames a channel from the context menu", async () => {
      const ch = await createChannel();
      const root = await createChannelGroup(ch);
      await renderChannelTree(root);
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
      const root = await createChannelGroup(ch);
      await renderChannelTree(root);
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

    it("renames the real channel name while an alias is shown", async () => {
      const ch = await createChannel();
      const root = await createChannelGroup(ch);
      const rng = await createTestRange(client);
      const { store } = await renderChannelTree(root);
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      const alias = uniqueName("alias");
      await client.ranges.setAlias(rng.key, ch.key, alias);
      await screen.findByText(alias);
      await openTreeRowContextMenu(alias);
      fireEvent.click(await screen.findByText("Rename"));
      // The row shows the alias, but the edit targets the channel's real name.
      const editor = await awaitTextEditingElement();
      expect(editor.innerText).toBe(ch.name);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.channels.retrieve(ch.key)).name).toBe(renamed),
      );
      // The alias comes back once the edit closes.
      expect(await screen.findByText(alias)).toBeTruthy();
    });

    it("sets and removes an alias under the active range", async () => {
      const ch = await createChannel();
      const root = await createChannelGroup(ch);
      const rng = await createTestRange(client);
      const { store } = await renderChannelTree(root);
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      await openTreeRowContextMenu(ch.name);
      fireEvent.click(await screen.findByText(`Set alias under ${rng.name}`));
      const editor = await awaitTextEditingElement();
      const alias = uniqueName("alias");
      commitTextEdit(editor, alias);
      await waitFor(async () =>
        expect(await client.ranges.retrieveAlias(rng.key, ch.key)).toBe(alias),
      );
      await openTreeRowContextMenu(alias);
      fireEvent.click(await screen.findByText(`Remove alias under ${rng.name}`));
      await waitFor(async () => {
        const aliases = await client.ranges.retrieveAliases(rng.key, [ch.key]);
        expect(aliases[ch.key]).toBeUndefined();
      });
    });

    it("opens the calculation editor for a calculated channel", async () => {
      const calc = await createChannel({
        isIndex: false,
        virtual: true,
        expression: "return 1",
      });
      const root = await createChannelGroup(calc);
      await renderChannelTree(root);
      await openTreeRowContextMenu(calc.name);
      fireEvent.click(await screen.findByText("Edit calculation"));
      expect(await screen.findByDisplayValue(calc.name)).toBeTruthy();
    });

    it("withholds rename and aliasing from an internal channel", async () => {
      const ch = await createChannel();
      const rng = await createTestRange(client);
      assertDefined(Item.ContextMenu);
      const { store } = await renderTreeContextMenu(Item.ContextMenu, {
        client,
        resources: [
          createResource(channelClient.ontologyID(ch.key), ch.name, {
            ...ch.payload,
            internal: true,
          }),
        ],
      });
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      // Delete shares the update permission the rename item needs, so its presence
      // proves the gate resolved before the absences below are read.
      expect(await screen.findByText("Delete")).toBeTruthy();
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText(/Set alias under/)).toBeNull();
    });
  });
});

describe("permission to write the channel", () => {
  it("should withhold rename, grouping, aliasing, and delete from a viewer", async () => {
    const ch = await createChannel();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [
        createResource(channelClient.ontologyID(ch.key), ch.name, { ...ch.payload }),
      ],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Group selection")).toBeNull();
    expect(screen.queryByText("Set Alias")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});
