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
  type ontology,
  project,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, uniqueName } from "@/testutil";

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

const renderSearchItem = async (resource: ontology.Resource) => {
  const Harness = (): ReactElement => {
    const staticProps = List.useStaticData<string, ontology.Resource>({
      data: [resource],
    });
    return (
      <Select.Frame<string, ontology.Resource>
        {...staticProps}
        value={undefined}
        onChange={() => {}}
      >
        <SearchListItem key={resource.key} itemKey={resource.key} index={0} />
      </Select.Frame>
    );
  };
  const { wrapper, store } = await createConsoleWrapper({ client });
  render(<Harness />, { wrapper });
  return store;
};

describe("channel/search", () => {
  it("creates a line plot for the selection when no plot is focused", async () => {
    const ch = await createChannel();
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const store = await renderSearchItem(
      createResource(channelClient.ontologyID(ch.key), ch.name),
    );
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe("lineplot");
    const plot = await client.lineplots.retrieve({ key: tab.resource.key });
    expect(plot.name).toBe("Line Plot");
    expect(plot.channels.y1).toContain(ch.key);
  });

  it("adds the selection to the focused line plot", async () => {
    const ch = await createChannel();
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const plot = await client.lineplots.create(proj.key, { name: uniqueName("plot") });
    const store = await renderSearchItem(
      createResource(channelClient.ontologyID(ch.key), ch.name),
    );
    const tabKey = uuid.create();
    const pan = await client.panels.create({
      name: uniqueName("panel"),
      parent: project.ontologyID(proj.key),
      root: {
        variant: "leaf",
        tabs: [
          { variant: "resource", key: tabKey, resource: lineplot.ontologyID(plot.key) },
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
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    await waitFor(async () => {
      const { channels } = await client.lineplots.retrieve({ key: plot.key });
      expect(channels.y1).toContain(ch.key);
    });
  });

  it("does not create a plot for a virtual channel without an expression", async () => {
    const ch = await createChannel({ isIndex: false, virtual: true });
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const store = await renderSearchItem(
      createResource(channelClient.ontologyID(ch.key), ch.name, {
        virtual: true,
        expression: "",
      }),
    );
    store.dispatch(Session.Project.select(proj.key));
    await act(async () => {
      fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    });
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
  });
});
