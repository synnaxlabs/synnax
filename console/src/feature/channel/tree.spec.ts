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
  createTestClient,
  DataType,
  group,
  NotFoundError,
  ontology,
} from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { placeLayout } from "@/platform/layout/testutil";
import { LinePlot } from "@/platform/lineplot";
import { findButton } from "@/platform/modals/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { createResource } from "@/platform/tree/testutil";
import {
  findTreeRow,
  openTreeRowContextMenu,
  renderOntologyTree,
} from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

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
    items: { channel: Channel.TREE_ITEM },
  });

describe("channel/ontology", () => {
  describe("haulItems", () => {
    it("hauls a regular channel as a channel item", async () => {
      const ch = await createChannel();
      const items = Channel.TREE_ITEM.haulItems(
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
      const items = Channel.TREE_ITEM.haulItems(
        createResource(channelClient.ontologyID(ch.key), ch.name, {
          internal: true,
        }),
      );
      expect(items).toHaveLength(1);
      expect(items[0].type).toBe("schematic-element");
    });
  });

  describe("onSelect", () => {
    it("creates a line plot for the selection when no plot is active", async () => {
      const ch = await createChannel();
      const proj = await client.projects.create({
        name: uniqueName("proj"),
        layout: {},
      });
      const root = await createChannelGroup(ch);
      const { store } = await renderChannelTree(root);
      store.dispatch(Session.Project.select(proj.key));
      fireEvent.doubleClick(await findTreeRow(ch.name));
      await waitFor(() => {
        const active = Session.Layout.selectActiveMosaicLayout(store.getState());
        expect(active?.type).toBe(LinePlot.LAYOUT_TYPE);
        expect(active?.name).toBe("Line Plot");
      });
    });

    it("adds the selection to the active line plot", async () => {
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
      placeLayout(store, plot.key, { type: LinePlot.LAYOUT_TYPE });
      fireEvent.doubleClick(await findTreeRow(ch.name));
      await waitFor(async () => {
        const { channels } = await client.lineplots.retrieve({ key: plot.key });
        expect(channels.y1).toContain(ch.key);
      });
    });

    it("does not create a plot for a virtual channel without an expression", async () => {
      const virtualCh = await createChannel({ isIndex: false, virtual: true });
      const realCh = await createChannel();
      const proj = await client.projects.create({
        name: uniqueName("proj"),
        layout: {},
      });
      const root = await createChannelGroup(virtualCh, realCh);
      const { store } = await renderChannelTree(root);
      store.dispatch(Session.Project.select(proj.key));
      fireEvent.doubleClick(await findTreeRow(virtualCh.name));
      fireEvent.doubleClick(await findTreeRow(realCh.name));
      await waitFor(() =>
        expect(Session.Layout.selectActiveMosaicLayout(store.getState())?.type).toBe(
          LinePlot.LAYOUT_TYPE,
        ),
      );
      const plots = Session.Layout.selectMany(store.getState()).filter(
        (l) => l.type === LinePlot.LAYOUT_TYPE,
      );
      expect(plots).toHaveLength(1);
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
  });
});
