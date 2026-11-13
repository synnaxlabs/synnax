// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  channel as channelClient,
  DataType,
  lineplot,
  log,
  panel,
  project,
  type resource,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

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

const renderSearchItem = async (item: resource.Resource, as: Synnax = client) => {
  const Harness = (): ReactElement => {
    const staticProps = List.useStaticData<string, resource.Resource>({
      data: [item],
    });
    return (
      <Select.Frame<string, resource.Resource>
        {...staticProps}
        value={undefined}
        onChange={() => {}}
      >
        <SearchListItem key={item.key} itemKey={item.key} index={0} />
      </Select.Frame>
    );
  };
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  render(<Harness />, { wrapper });
  return store;
};

// Opens a log in a panel of a new project and focuses its tab.
const focusLog = async (store: TestStore): Promise<log.Key> => {
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const doc = await client.logs.create({ project: proj.key, name: uniqueName("log") });
  const tabKey = uuid.create();
  const pan = await client.panels.create({
    name: uniqueName("panel"),
    project: proj.key,
    root: {
      variant: "leaf",
      tabs: [{ variant: "resource", key: tabKey, resource: log.resourceID(doc.key) }],
    },
  });
  store.dispatch(Session.Project.select(proj.key));
  store.dispatch(Session.Panel.select({ key: pan.key }));
  store.dispatch(
    Session.Panel.internalSelectTab({ key: pan.key, tabKey, otherTabKeys: [tabKey] }),
  );
  return doc.key;
};

const channelsOf = async (key: log.Key): Promise<channelClient.Key[]> =>
  (await client.logs.retrieve(key)).channels.map((e) => e.channel);

describe("channel/search", () => {
  it("creates a line plot for the selection when no plot is focused", async () => {
    const ch = await createChannel();
    const proj = await client.projects.create({
      name: uniqueName("proj"),
    });
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
    );
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
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
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
    );
    const tabKey = uuid.create();
    const pan = await client.panels.create({
      name: uniqueName("panel"),
      project: proj.key,
      root: {
        variant: "leaf",
        tabs: [
          { variant: "resource", key: tabKey, resource: lineplot.resourceID(plot.key) },
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
      const { channels } = await client.lineplots.retrieve(plot.key);
      expect(channels.y1).toContain(ch.key);
    });
  });

  it("creates a log for a virtual channel without an expression", async () => {
    const ch = await createChannel({ isIndex: false, virtual: true });
    const proj = await client.projects.create({
      name: uniqueName("proj"),
    });
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
    );
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe("log");
    const opened = await client.logs.retrieve(tab.resource.key);
    expect(opened.name).toBe("Log");
    expect(opened.channels.map((e) => e.channel)).toContain(ch.key);
  });

  it("adds a virtual channel to the focused log", async () => {
    const ch = await createChannel({ isIndex: false, virtual: true });
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
    );
    const key = await focusLog(store);
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    await waitFor(async () => expect(await channelsOf(key)).toContain(ch.key));
  });

  it("adds a persisted channel to the focused log", async () => {
    const ch = await createChannel();
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
    );
    const key = await focusLog(store);
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    await waitFor(async () => expect(await channelsOf(key)).toContain(ch.key));
  });

  it("opens a line plot when the user cannot edit the focused log", async () => {
    const ch = await createChannel();
    const editor = await createTestClientWithPolicy(client, [
      {
        name: uniqueName("policy"),
        objects: [
          channelClient.TYPE_RESOURCE_ID,
          log.TYPE_RESOURCE_ID,
          user.TYPE_RESOURCE_ID,
          access.role.TYPE_RESOURCE_ID,
          access.policy.TYPE_RESOURCE_ID,
        ],
        actions: ["retrieve"],
      },
      {
        name: uniqueName("policy"),
        objects: [
          lineplot.TYPE_RESOURCE_ID,
          panel.TYPE_RESOURCE_ID,
          project.TYPE_RESOURCE_ID,
        ],
        actions: ["retrieve", "create", "update"],
      },
    ]);
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
      editor,
    );
    const key = await focusLog(store);
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    const tab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "resource" && t.resource.type === "lineplot",
    );
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    const plot = await client.lineplots.retrieve(tab.resource.key);
    expect(plot.channels.y1).toContain(ch.key);
    expect(await channelsOf(key)).toEqual([]);
  });

  it("creates a line plot for a calculated channel", async () => {
    const ch = await createChannel({
      isIndex: false,
      virtual: true,
      expression: "return 1",
    });
    const proj = await client.projects.create({
      name: uniqueName("proj"),
    });
    const store = await renderSearchItem(
      createResource(channelClient.resourceID(ch.key), ch.name),
    );
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText(ch.name), { detail: 0 });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe("lineplot");
    const plot = await client.lineplots.retrieve(tab.resource.key);
    expect(plot.channels.y1).toContain(ch.key);
  });
});
