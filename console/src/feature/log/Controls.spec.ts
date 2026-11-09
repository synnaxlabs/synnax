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
import { act, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Log } from "@/feature/log/Log";
import { client, renderLog } from "@/feature/log/toolbar/testutil";
import { Session } from "@/session";

const stateZ = z.looseObject({});

// The real worker needs a canvas. This one reports lines that streamed in.
const createWorker = (newLines: number): aether.ComponentRegistry => {
  class Worker extends aether.Leaf<typeof stateZ> {
    static readonly TYPE = "log";
    schema = stateZ;

    afterUpdate(): void {
      if (this.state.newLines === newLines) return;
      this.setState((s) => ({ ...s, empty: false, newLines }));
    }
  }
  return { [Worker.TYPE]: Worker };
};

const renderControls = async (newLines: number) => {
  const ch = await client.channels.create({
    name: id.create(),
    dataType: DataType.STRING,
    virtual: true,
  });
  return await renderLog(Log, {
    additionalRegistry: createWorker(newLines),
    log: {
      channels: [log.channelEntryZ.parse({ channel: ch.key, color: color.ZERO })],
    },
  });
};

describe("log/Controls", () => {
  it("should show the count of new lines while the log is paused", async () => {
    const { key, store } = await renderControls(3);
    act(() => {
      store.dispatch(Session.Log.setHold({ key, hold: true }));
    });
    expect(await screen.findByText("3 new lines")).toBeDefined();
  });

  it("should name a single new line", async () => {
    const { key, store } = await renderControls(1);
    act(() => {
      store.dispatch(Session.Log.setHold({ key, hold: true }));
    });
    expect(await screen.findByText("1 new line")).toBeDefined();
  });

  it("should hide the count when the log resumes", async () => {
    const { key, store } = await renderControls(3);
    act(() => {
      store.dispatch(Session.Log.setHold({ key, hold: true }));
    });
    await screen.findByText("3 new lines");
    act(() => {
      store.dispatch(Session.Log.setHold({ key, hold: false }));
    });
    expect(screen.queryByText("3 new lines")).toBeNull();
  });
});
