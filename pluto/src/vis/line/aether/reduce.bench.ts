// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import {
  type Aggregation,
  dynamicWindow,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterAll, test } from "vitest";

import { ChannelData, type ChannelDataProps } from "@/telem/aether/remote";
import { telemTest } from "@/telem/aether/testutil";
import {
  buildDrawOperations,
  DEFAULT_OVERLAP_THRESHOLD,
  type DrawOperation,
} from "@/vis/line/aether/draw";
import { groupSize, Reducer, type View } from "@/vis/line/aether/reduce";

// Reduces the full-resolution samples of a live line 1000 pixels wide that shows the
// last 10 minutes of a 1 kHz channel, as the line does once per frame.

const START = TimeStamp.seconds(1_750_000_000);
const SPAN = TimeSpan.minutes(10);
const FRAME = TimeSpan.milliseconds(16);
const GROUPS_PER_COLUMN = 1 / 2;

const core = new SyntheticCore({
  rate: 1000,
  start: START,
  now: START.add(TimeSpan.hours(1)),
});
const client = telemTest.createClient(core);
afterAll(async () => await client.feed.close());

const view = (): View => ({
  bounds: new TimeRange(core.now().sub(SPAN), core.now()).numericBounds,
  width: 1000,
});

const createSource = (props: ChannelDataProps): ChannelData =>
  new ChannelData(client, props, client.now);

const channel = core.data[0].key;
const x = createSource({
  channel,
  useIndexOfChannel: true,
  window: dynamicWindow(SPAN),
});
const y = createSource({ channel, window: dynamicWindow(SPAN) });

const read = (): DrawOperation[] => {
  const { bounds, width } = view();
  const v = { bounds, points: width * GROUPS_PER_COLUMN * 2 };
  return buildDrawOperations(
    x.value(v).data[1],
    y.value(v).data[1],
    DEFAULT_OVERLAP_THRESHOLD,
  );
};

const deadline = TimeStamp.now().add(TimeSpan.seconds(30));
while ([x, y].some((s) => s.value().status.variant === "loading")) {
  if (TimeStamp.now().after(deadline)) throw new Error("sources did not settle");
  read();
  await sleep.sleep(TimeSpan.milliseconds(1));
}

let ops = read();
const size = groupSize(ops, view(), GROUPS_PER_COLUMN);

const cold = (aggregation: Aggregation) => (): void =>
  void new Reducer().reduce(ops, aggregation, size);

const warm = new Reducer();

test("reduce a 10 minute live window of 1 kHz samples", async ({ bench }) => {
  await bench.compare(
    bench("group size", () => void groupSize(ops, view(), GROUPS_PER_COLUMN)),
    bench("min_max, no reduced pairs held", cold("min_max")),
    bench("average, no reduced pairs held", cold("average")),
    bench("decimate, no reduced pairs held", cold("decimate")),
    bench(
      "min_max, every reduced pair held",
      { beforeAll: () => void warm.reduce(ops, "min_max", size) },
      () => void warm.reduce(ops, "min_max", size),
    ),
    // Each run reduces 16 ms of new samples, which the stream lands before it.
    bench(
      "min_max, 16 ms of new samples",
      {
        beforeEach: async () => {
          core.advance(FRAME);
          await new Promise((resolve) => setImmediate(resolve));
          ops = read();
        },
      },
      () => void warm.reduce(ops, "min_max", size),
    ),
  );
});
