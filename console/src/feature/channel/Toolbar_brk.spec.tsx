// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { HTTPClient } from "@synnaxlabs/freighter";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { Channel as PChannel } from "@synnaxlabs/pluto";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, render, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { Channel } from "@/feature/channel";
import { Tree } from "@/platform/tree";
import { renderToolbar } from "@/platform/tree/testutil";
import { CONNECTION_PARAMS } from "@/session/core/testutil";
import { CaptureStatuses, createConnectedConsoleWrapper } from "@/testutil";

const BUILT_INS_TARGET = "/channel/retrieve-built-ins";

describe("channel/Toolbar built-in keys failure report", () => {
  it("should not report the failure again when the toolbar reopens and the keys arrive", async () => {
    const client = createTestClient({ onInternalError: () => {} });
    await client.connect();
    await waitForStreamLive(client.connection);
    await client.channels.create(
      [0, 1].map(() => ({
        name: uniqueName("ch"),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      })),
    );
    const fault = { on: true };
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (fault.on && args[0] === BUILT_INS_TARGET) throw new Error("refused");
      return await send(...args);
    });
    let setShown: (shown: boolean) => void = () => {};
    const Drawer = (): ReactElement | null => {
      const [shown, set] = useState(true);
      setShown = set;
      return shown ? Channel.TOOLBAR.content : null;
    };
    let statuses: Status.NotificationSpec[] = [];
    await renderToolbar(
      <>
        <Drawer />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
      { client, items: Channel.TREE_ITEMS },
    );
    const reports = () =>
      statuses
        .filter(({ message }) => message === "Failed to retrieve built-in channels")
        .reduce((n, { count }) => n + count, 0);
    await waitFor(() => expect(reports()).toBe(1));
    // The Core answers again. The user closes the channel drawer and opens it.
    fault.on = false;
    await act(async () => setShown(false));
    await act(async () => setShown(true));
    await waitFor(() => expect(PChannel.getCachedBuiltIns(client)).toBeDefined());
    await act(async () => {});
    expect(reports()).toBe(1);
  });

  it("should show one error status for one failed fetch under the real provider", async () => {
    const seed = createTestClient();
    await seed.channels.create(
      [0, 1].map(() => ({
        name: uniqueName("ch"),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      })),
    );
    const original = HTTPClient.prototype.send;
    const spy = vi
      .spyOn(HTTPClient.prototype, "send")
      .mockImplementation(async function (this: HTTPClient, ...args) {
        if (args[0] === BUILT_INS_TARGET) throw new Error("refused");
        return await original.apply(this, args);
      });
    onTestFinished(() => spy.mockRestore());
    // The status handler logs each error it shows.
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { wrapper } = await createConnectedConsoleWrapper({
      client: createTestClient({ onInternalError: () => {} }),
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
    await waitFor(
      () =>
        expect(
          statuses.some(
            ({ message }) => message === "Failed to retrieve built-in channels",
          ),
        ).toBe(true),
      { timeout: 10000 },
    );
    await new Promise((resolve) => setTimeout(resolve, 2000));
    await act(async () => {});
    expect(
      statuses
        .filter(({ variant }) => variant === "error")
        .map(({ message, description, count }) => ({ message, description, count })),
    ).toEqual([
      {
        message: "Failed to retrieve built-in channels",
        description: "Failed to retrieve built-in channels: refused",
        count: 1,
      },
    ]);
  }, 30000);
});
