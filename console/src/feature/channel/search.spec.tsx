// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel as channelClient, DataType, type ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { placeLayout } from "@/platform/layout/testutil";
import { LinePlot } from "@/platform/lineplot";
import { createEntry } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createChannel = async (overrides: Partial<channelClient.New> = {}) =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
    ...overrides,
  });

const SearchListItem = Channel.SEARCH_LIST_ITEMS.channel;
if (SearchListItem == null) throw new Error("channel SearchListItem is not defined");

const renderSearchItems = async (resources: ontology.Resource[]) => {
  const Harness = (): ReactElement => {
    const staticProps = List.useStaticData<string, ontology.Resource>({
      data: resources,
    });
    return (
      <Select.Frame<string, ontology.Resource>
        {...staticProps}
        value={undefined}
        onChange={() => {}}
      >
        {resources.map((resource, i) => (
          <SearchListItem key={resource.key} itemKey={resource.key} index={i} />
        ))}
      </Select.Frame>
    );
  };
  const { wrapper, store } = await createConsoleWrapper({ client });
  render(<Harness />, { wrapper });
  return store;
};

describe("channel/search", () => {
  it("creates a line plot for the selection when no plot is active", async () => {
    const ch = await createChannel();
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const store = await renderSearchItems([
      createEntry(channelClient.ontologyID(ch.key), ch.name),
    ]);
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
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
    const plot = await client.lineplots.create(proj.key, { name: uniqueName("plot") });
    const store = await renderSearchItems([
      createEntry(channelClient.ontologyID(ch.key), ch.name),
    ]);
    placeLayout(store, plot.key, { type: LinePlot.LAYOUT_TYPE });
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
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
    const store = await renderSearchItems([
      createEntry(channelClient.ontologyID(virtualCh.key), virtualCh.name),
      createEntry(channelClient.ontologyID(realCh.key), realCh.name),
    ]);
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText(virtualCh.name), { detail: 0 });
    fireEvent.click(await screen.findByText(realCh.name), { detail: 0 });
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
