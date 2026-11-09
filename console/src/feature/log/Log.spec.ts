// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, log } from "@synnaxlabs/client";
import { aether } from "@synnaxlabs/pluto/ether";
import { color, id } from "@synnaxlabs/x";
import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Log } from "@/feature/log/Log";
import { client, renderLog } from "@/feature/log/toolbar/testutil";
import { Session } from "@/session";

const stateZ = z.looseObject({ hiddenChannels: z.array(z.number()).optional() });

// The real worker needs a canvas. The log renders its empty content without one.
class WorkerStub extends aether.Leaf<typeof stateZ> {
  static readonly TYPE = "log";
  schema = stateZ;
}

const REGISTRY = { [WorkerStub.TYPE]: WorkerStub };

describe("log/Log", () => {
  it("asks for channels when the log has none", async () => {
    await renderLog(Log, { additionalRegistry: REGISTRY });
    expect(await screen.findByText("No channels configured")).toBeDefined();
    expect(screen.getByText("Configure channels")).toBeDefined();
  });

  it("waits for data when the log has a channel", async () => {
    const ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    await renderLog(Log, {
      additionalRegistry: REGISTRY,
      log: {
        channels: [log.channelEntryZ.parse({ channel: ch.key, color: color.ZERO })],
      },
    });
    expect(await screen.findByText("No data received yet")).toBeDefined();
    expect(screen.queryByText("Configure channels")).toBeNull();
  });

  it("names a filter that matches nothing", async () => {
    const ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const { key, store } = await renderLog(Log, {
      additionalRegistry: REGISTRY,
      log: {
        channels: [log.channelEntryZ.parse({ channel: ch.key, color: color.ZERO })],
      },
    });
    act(() => {
      store.dispatch(Session.Log.setSearch({ key, text: "err", filtered: true }));
    });
    expect(await screen.findByText("No matching lines")).toBeDefined();
    act(() => {
      store.dispatch(Session.Log.setSearch({ key, text: "(", mode: "regex" }));
    });
    expect(await screen.findByText("No data received yet")).toBeDefined();
  });

  it("hands the hidden channels of the session to the log", async () => {
    const hidden: Array<number[] | undefined> = [];
    class Recorder extends WorkerStub {
      afterUpdate(): void {
        hidden.push(this.state.hiddenChannels);
      }
    }
    const { key, store } = await renderLog(Log, {
      additionalRegistry: { [Recorder.TYPE]: Recorder },
    });
    act(() => {
      store.dispatch(
        Session.Log.setChannelVisible({ key, channel: 7, visible: false }),
      );
    });
    await waitFor(() => expect(hidden.at(-1)).toEqual([7]));
  });
});
