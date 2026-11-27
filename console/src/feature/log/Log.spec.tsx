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
import { aether, canvas } from "@synnaxlabs/pluto/ether";
import { id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Log } from "@/feature/log";
import { client, renderLog } from "@/feature/log/toolbar/testutil";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { installMissingGPU, type TestStore } from "@/testutil";

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
  override schema = stateZ;
  override methods = methodsZ;

  seek(time: string): void {
    seeks(time);
  }
}

const REGISTRY = { [WorkerStub.TYPE]: WorkerStub };

const { Content } = Log.TABS[log.TYPE_RESOURCE_ID.type];

beforeAll(Content.preload);

describe("Log tab content", () => {
  it("asks for channels when the log has none", async () => {
    await renderLog(Content, { additionalRegistry: REGISTRY });
    expect(await screen.findByText("No channels configured")).toBeDefined();
    expect(screen.getByText("Configure channels")).toBeDefined();
  });

  it("waits for data when the log has a channel", async () => {
    const ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    await renderLog(Content, {
      additionalRegistry: REGISTRY,
      log: {
        channels: [log.channelEntryZ.parse({ channel: ch.key })],
      },
    });
    expect(await screen.findByText("No data received yet")).toBeDefined();
    expect(screen.queryByText("Configure channels")).toBeNull();
  });

  describe("when the canvas has no GPU", () => {
    installMissingGPU();

    it("shows a placeholder in place of the log", async () => {
      const InCanvas = () => (
        <Vis.Canvas>
          <Content />
        </Vis.Canvas>
      );
      await renderLog(InCanvas, { additionalRegistry: canvas.REGISTRY });
      expect(await screen.findByText("GPU required")).toBeDefined();
      expect(
        screen.getByText(
          "Logs need a GPU to render. Run the Console on a machine with a GPU.",
        ),
      ).toBeDefined();
      expect(screen.queryByText("No channels configured")).toBeNull();
    });
  });

  describe("range drop", () => {
    const RANGE = new TimeRange(TimeStamp.seconds(10), TimeStamp.MAX);
    const DRAG_HANDLE = "drag-a-range";

    const DragSource = (): ReactElement => {
      const { startDrag, onDragEnd } = Haul.useDrag({ type: "test-source" });
      const items = [
        {
          type: Ranger.HAUL_TYPE,
          key: "range",
          data: { key: "range", name: "Range", timeRange: RANGE.numeric },
        },
      ];
      return (
        <button
          onDragStart={(event) => startDrag({ event, items, preview: null })}
          onDragEnd={onDragEnd}
        >
          {DRAG_HANDLE}
        </button>
      );
    };

    const WithSource = (): ReactElement => (
      <Haul.Provider>
        <DragSource />
        <Content />
      </Haul.Provider>
    );

    const dropRange = async () => {
      seeks.mockClear();
      const rendered = await renderLog(WithSource, { additionalRegistry: REGISTRY });
      const target = await screen.findByText("No channels configured");
      fireEvent.dragStart(screen.getByText(DRAG_HANDLE));
      fireEvent.drop(target);
      return rendered;
    };

    const docOf = (store: TestStore, key: string) =>
      documentIn(store.getState()[Session.Log.SLICE_NAME], key);

    const DROPPED = {
      start: RANGE.start.valueOf().toString(),
      end: RANGE.end.valueOf().toString(),
    };

    it("goes to the start of a dropped range, paused, with the range kept", async () => {
      const { key, store } = await dropRange();
      await waitFor(() => expect(seeks).toHaveBeenCalledWith(DROPPED.start));
      expect(docOf(store, key)).toMatchObject({ hold: true, range: DROPPED });
    });

    it("clears the range when the log resumes", async () => {
      const { key, store } = await dropRange();
      await waitFor(() => expect(docOf(store, key)?.range).toEqual(DROPPED));
      act(() => void store.dispatch(Session.Log.setHold({ key, hold: false })));
      expect(docOf(store, key)?.range).toBeNull();
    });
  });
});
