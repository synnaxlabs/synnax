// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  failRequests,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { HTTPClient } from "@synnaxlabs/freighter";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
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
  it("should neither fetch nor report again when the toolbar reopens", async () => {
    const internal: Error[] = [];
    const client = createTestClient({ onInternalError: (e) => internal.push(e) });
    await client.connect();
    await waitForStreamLive(client.connection);
    await client.channels.create(
      [0, 1].map(() => ({
        name: uniqueName("ch"),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      })),
    );
    failRequests(client, BUILT_INS_TARGET, new Error("refused"));
    const send = spyOnSend(client);
    const fetches = () =>
      send.mock.calls.filter(([target]) => target === BUILT_INS_TARGET).length;
    let setShown: (shown: boolean) => void = () => {};
    const Drawer = (): ReactElement | null => {
      const [shown, set] = useState(true);
      setShown = set;
      return shown ? Channel.TOOLBAR.content : null;
    };
    await renderToolbar(<Drawer />, { client, items: Channel.TREE_ITEMS });
    await waitFor(() =>
      expect(internal.map((e) => e.message)).toEqual([
        "failed to fetch built-in channels",
      ]),
    );
    expect(fetches()).toBe(1);
    // The user closes the channel drawer and opens it.
    await act(async () => setShown(false));
    await act(async () => setShown(true));
    await act(async () => {});
    expect(internal).toHaveLength(1);
    expect(fetches()).toBe(1);
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
    // Every client the providers open sends through the prototype.
    const original = Object.getOwnPropertyDescriptor(HTTPClient.prototype, "send")
      ?.value as HTTPClient["send"];
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
    await new Promise((resolve) => setTimeout(resolve, 2000));
    await act(async () => {});
    expect(errors()).toEqual([
      {
        message: "Failed to sync with Core",
        description: "failed to fetch built-in channels: refused",
        count: 1,
      },
    ]);
  }, 30000);
});
