// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, channel, DataType, query, Synnax, user } from "@synnaxlabs/client";
import {
  createCuttingProxy,
  createProxiedTestClient,
  createTestClient,
  createTestClientWithPolicy,
  failRequests,
  FAST_RETRY,
  rejectRequests,
  spyOnSend,
  testClientParams,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { HTTPClient } from "@synnaxlabs/freighter";
import { Haul } from "@synnaxlabs/lyra/haul";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { Channel } from "@/feature/channel";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  HOST_ID,
  renderLiveTree,
  renderToolbar,
} from "@/platform/tree/testutil";
import { CONNECTION_PARAMS } from "@/session/core/testutil";
import { CaptureStatuses, createConnectedConsoleWrapper } from "@/testutil";

const client = createTestClient();

const TARGET = "/channel/retrieve-built-ins";

const REPORT = "Failed to fetch built-in channels. refused";

const createIndex = async () =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const createIndexes = async () =>
  await client.channels.create(
    [0, 1].map(() => ({
      name: uniqueName("ch"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    })),
  );

const hostChannels = (...channels: { key: channel.Key }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.channels,
      query.keysRequest(channels.map((ch) => ch.key)),
      (ch) => Tree.toResource("channel")(ch.payload),
    ),
  ]);

const editable = (name: string) =>
  expect(screen.getByText(name).className).toContain("lyra-text--editable");

describe("channel/tree built-in keys", () => {
  it("should keep the name field editable after a reconnect follows a failed fetch", async () => {
    const ch = await createIndex();
    const { proxy, client: proxied } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onInternalError: () => {},
    });
    await proxied.connect();
    await waitForStreamLive(proxied.connection);
    const fault = { on: true };
    rejectRequests(
      proxied,
      (target) => fault.on && target === TARGET,
      new Error("refused"),
    );
    await renderLiveTree({
      client: proxied,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    // A failed fetch leaves the rename enabled.
    await waitFor(() => editable(ch.name));
    // The Core answers again, and the connection drops and comes back.
    fault.on = false;
    const epoch = proxied.connection.status.details.epoch;
    await proxy.sever();
    await proxy.restore();
    await waitFor(
      () => expect(proxied.connection.status.details.epoch).toBeGreaterThan(epoch),
      { timeout: 15000 },
    );
    // Another user renames the channel, so its tree item renders again.
    const renamed = uniqueName("renamed");
    await client.channels.rename(ch.key, renamed);
    await screen.findByText(renamed, undefined, { timeout: 8000 });
    await waitFor(() => editable(renamed), { timeout: 8000 });
  }, 60000);

  it("should make a channel renamable after one cut response while the stream stays live", async () => {
    const ch = await createIndex();
    const proxy = await createCuttingProxy();
    const proxied = new Synnax({
      ...testClientParams(),
      host: "localhost",
      port: proxy.port,
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: () => {},
    });
    onTestFinished(async () => await proxied.close());
    await proxied.connect();
    await waitForStreamLive(proxied.connection);
    const epoch = proxied.connection.status.details.epoch;
    proxy.cutNext(TARGET);
    await renderLiveTree({
      client: proxied,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    await waitFor(() => expect(proxy.cuts()).toBe(1));
    await waitFor(() => editable(ch.name));
    expect(proxied.connection.status.details.streamLive).toBe(true);
    expect(proxied.connection.status.details.epoch).toBe(epoch);
  }, 30000);

  it("should make a channel renamable for a user whose stream the Core refuses", async () => {
    const ch = await createIndex();
    // The user can read and rename the channel, but not read the change channels.
    const restricted = await createTestClientWithPolicy(client, [
      {
        name: uniqueName("policy"),
        objects: [
          user.TYPE_RESOURCE_ID,
          access.role.TYPE_RESOURCE_ID,
          access.policy.TYPE_RESOURCE_ID,
          channel.resourceID(ch.key),
        ],
        actions: ["retrieve"],
      },
      {
        name: uniqueName("policy"),
        objects: [channel.resourceID(ch.key)],
        actions: ["update"],
      },
    ]);
    const status = await restricted.connect();
    expect(status.details.streamDenied).toBe(true);
    await renderLiveTree({
      client: restricted,
      projectClient: client,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    await screen.findByText(ch.name);
    await waitFor(() => editable(ch.name));
    expect(restricted.channels.builtIns.state().keys?.size).toBeGreaterThan(0);
  }, 30000);
});

describe("channel/Toolbar built-in keys", () => {
  it("should not report a failure again when the toolbar reopens", async () => {
    const internal: Error[] = [];
    const reporting = createTestClient({ onInternalError: (e) => internal.push(e) });
    await reporting.connect();
    await waitForStreamLive(reporting.connection);
    await createIndexes();
    failRequests(reporting, TARGET, new Error("refused"));
    const send = spyOnSend(reporting);
    const fetches = () =>
      send.mock.calls.filter(([target]) => target === TARGET).length;
    let setShown: (shown: boolean) => void = () => {};
    const Drawer = (): ReactElement | null => {
      const [shown, set] = useState(true);
      setShown = set;
      return shown ? Channel.TOOLBAR.content : null;
    };
    await renderToolbar(<Drawer />, { client: reporting, items: Channel.TREE_ITEMS });
    await waitFor(() => expect(internal.map((e) => e.message)).toEqual([REPORT]));
    // The user closes the channel drawer and opens it, which fetches again.
    const before = fetches();
    await act(async () => setShown(false));
    await act(async () => setShown(true));
    await waitFor(() => expect(fetches()).toBeGreaterThan(before));
    await act(async () => await reporting.channels.builtIns.retrieve());
    expect(internal.map((e) => e.message)).toEqual([REPORT]);
  });

  it("should show one error status for one failed fetch under the real provider", async () => {
    await createIndexes();
    // Every client the providers open sends through the prototype.
    const original = Object.getOwnPropertyDescriptor(HTTPClient.prototype, "send")
      ?.value as HTTPClient["send"];
    const sent: string[] = [];
    const spy = vi
      .spyOn(HTTPClient.prototype, "send")
      .mockImplementation(async function (this: HTTPClient, ...args) {
        sent.push(args[0]);
        if (args[0] === TARGET) throw new Error("refused");
        return await original.apply(this, args);
      });
    onTestFinished(() => spy.mockRestore());
    // The status handler logs each error it shows.
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { wrapper } = await createConnectedConsoleWrapper({
      client: createTestClient(),
      connParams: CONNECTION_PARAMS,
    });
    let statuses: Status.NotificationSpec[] = [];
    render(
      <>
        <Haul.Provider>
          <Tree.Provider items={Channel.TREE_ITEMS}>
            {Channel.TOOLBAR.content}
          </Tree.Provider>
        </Haul.Provider>
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
      { wrapper },
    );
    const errors = () =>
      statuses
        .filter(({ variant }) => variant === "error")
        .map(({ message, description, count }) => ({ message, description, count }));
    await waitFor(() => expect(errors()).not.toEqual([]), { timeout: 10000 });
    // Every tree item has mounted and fetched by the time the rows show.
    await waitFor(() =>
      expect(document.querySelectorAll(".lyra-tree__item").length).toBeGreaterThan(1),
    );
    await waitFor(() =>
      expect(sent.filter((t) => t === TARGET).length).toBeGreaterThan(0),
    );
    await act(async () => {});
    expect(errors()).toEqual([
      {
        message: "Failed to sync with Core",
        description: REPORT,
        count: 1,
      },
    ]);
  }, 30000);
});
