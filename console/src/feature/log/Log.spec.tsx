// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, log } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Ranger } from "@synnaxlabs/pluto";
import { aether } from "@synnaxlabs/pluto/ether";
import { color, id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Log } from "@/feature/log/Log";
import { client, renderLog } from "@/feature/log/toolbar/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { type TestStore } from "@/testutil";

const stateZ = z.looseObject({});
const methodsZ = {
  seek: z.function({ input: z.tuple([z.string()]), output: z.void() }),
};

const seeks = vi.fn();

// The real worker needs a canvas. The log renders its empty content without one.
class WorkerStub
  extends aether.Leaf<typeof stateZ, {}, typeof methodsZ>
  implements aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "log";
  schema = stateZ;
  methods = methodsZ;

  seek(time: string): void {
    seeks(time);
  }
}

const REGISTRY = { [WorkerStub.TYPE]: WorkerStub };

// Holds entries, so the log shows its controls once the worker has its state.
class FilledStub extends WorkerStub {
  afterUpdate(): void {
    if (this.state.empty !== false) this.setState((s) => ({ ...s, empty: false }));
  }
}

const FILLED_REGISTRY = { [FilledStub.TYPE]: FilledStub };

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

  describe("range drop", () => {
    const RANGE = new TimeRange(TimeStamp.seconds(10), TimeStamp.MAX);
    const DRAG_HANDLE = "drag-a-range";

    const DragSource = (): ReactElement => {
      const { startDrag, onDragEnd } = Haul.useDrag({ type: "test-source" });
      const handleDragStart = (): void =>
        startDrag([
          {
            type: Ranger.HAUL_TYPE,
            key: "range",
            data: { key: "range", name: "Range", timeRange: RANGE.numeric },
          },
        ]);
      return (
        <button onDragStart={handleDragStart} onDragEnd={onDragEnd}>
          {DRAG_HANDLE}
        </button>
      );
    };

    const WithSource = (): ReactElement => (
      <>
        <DragSource />
        <Log />
      </>
    );

    const dropRange = async () => {
      seeks.mockClear();
      const rendered = await renderLog(WithSource, {
        additionalRegistry: FILLED_REGISTRY,
      });
      const target = await screen.findByLabelText("Go to a time");
      fireEvent.dragStart(screen.getByText(DRAG_HANDLE));
      fireEvent.drop(target);
      return rendered;
    };

    const docOf = (store: TestStore, key: string) =>
      documentIn(store.getState()[Session.Log.SLICE_NAME], key);

    it("goes to the start of a dropped range, paused, with the range kept", async () => {
      const { key, store } = await dropRange();
      await waitFor(() =>
        expect(seeks).toHaveBeenCalledWith(RANGE.start.valueOf().toString()),
      );
      expect(docOf(store, key)).toMatchObject({
        hold: true,
        range: {
          start: RANGE.start.valueOf().toString(),
          end: RANGE.end.valueOf().toString(),
        },
      });
    });

    it("clears the range when the log resumes", async () => {
      const { key, store } = await dropRange();
      await waitFor(() => expect(docOf(store, key)?.range).not.toBeNull());
      act(() => void store.dispatch(Session.Log.setHold({ key, hold: false })));
      expect(docOf(store, key)?.range).toBeNull();
    });
  });
});
