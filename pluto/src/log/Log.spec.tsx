// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { id, uuid } from "@synnaxlabs/x";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, Suspense } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { aether } from "@/aether/aether";
import { createHaulItem } from "@/channel/types";
import { Log } from "@/log";
import { log } from "@/log/aether";
import { createAsyncSynnaxWrapper } from "@/testutil";

// Stands in for the worker-side log and keeps each state the Log sends it.
const createRecorder = () => {
  const states: unknown[] = [];
  class Recorder extends aether.Leaf<typeof log.logStateZ> {
    static readonly TYPE = log.Log.TYPE;
    schema = log.logStateZ;

    afterUpdate(): void {
      states.push(this.state);
    }
  }
  return { registry: { [Recorder.TYPE]: Recorder }, last: () => states.at(-1) };
};

const client = createTestClient();

const DRAG_HANDLE = "Drag channel";

const DragSource = ({ channelKey }: { channelKey: channel.Key }): ReactElement => {
  const { startDrag, onDragEnd } = Haul.useDrag({ type: "test-source" });
  return (
    <button
      onDragStart={() => startDrag([createHaulItem(channelKey)])}
      onDragEnd={onDragEnd}
    >
      {DRAG_HANDLE}
    </button>
  );
};

describe("Log", () => {
  let wrapper: FC<PropsWithChildren>;
  let recorder: ReturnType<typeof createRecorder>;
  beforeEach(async () => {
    recorder = createRecorder();
    const Synnax = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: recorder.registry,
    });
    wrapper = ({ children }) => (
      <Synnax>
        <Triggers.Provider>
          <Haul.Provider>{children}</Haul.Provider>
        </Triggers.Provider>
      </Synnax>
    );
  });

  it("should send the stored display settings to the log", async () => {
    const project = await client.projects.create({ name: `log_${uuid.create()}` });
    const created = await client.logs.create({
      project: project.key,
      name: "display",
      timestampsHidden: true,
      channelNamesHidden: true,
      timestampPrecision: 2,
    });
    await act(async () => {
      render(
        <Suspense>
          <Log.Suspended logKey={created.key}>
            <Log.Log />
          </Log.Suspended>
        </Suspense>,
        { wrapper },
      );
    });
    await waitFor(() =>
      expect(recorder.last()).toMatchObject({
        timestampsHidden: true,
        channelNamesHidden: true,
        timestampPrecision: 2,
        channels: [],
      }),
    );
  });
  describe("channel drop", () => {
    const dropChannel = async (editable: boolean) => {
      const project = await client.projects.create({ name: `log_${uuid.create()}` });
      const created = await client.logs.create({ project: project.key, name: "drop" });
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await act(async () => {
        render(
          <Suspense>
            <DragSource channelKey={ch.key} />
            <Log.Suspended logKey={created.key}>
              <Log.Log editable={editable} />
            </Log.Suspended>
          </Suspense>,
          { wrapper },
        );
      });
      fireEvent.dragStart(await screen.findByText(DRAG_HANDLE));
      fireEvent.drop(await screen.findByText("No log entries"));
      return { key: created.key, channel: ch.key };
    };

    it("should add a dropped channel to the log", async () => {
      const { key, channel } = await dropChannel(true);
      await waitFor(async () => {
        const { channels } = await client.logs.retrieve(key);
        expect(channels.map((e) => e.channel)).toEqual([channel]);
      });
    });

    it("should not add a dropped channel to a log that is not editable", async () => {
      const { key } = await dropChannel(false);
      await expectAlways(async () => {
        expect((await client.logs.retrieve(key)).channels).toEqual([]);
      });
    });
  });
});
