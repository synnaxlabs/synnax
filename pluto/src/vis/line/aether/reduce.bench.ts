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
  DataType,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterAll, test } from "vitest";

import { TiledChannelData, type TiledChannelDataProps } from "@/telem/aether/remote";
import { type View } from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/test";
import {
  buildDrawOperations,
  DEFAULT_OVERLAP_THRESHOLD,
  type DrawOperation,
} from "@/vis/line/aether/line";
import { groupSize, Reducer } from "@/vis/line/aether/reduce";

// Reduces the full-resolution samples of a live line 1000 pixels wide that shows the
// last 10 minutes of a 1 kHz channel, as the line does once per frame.

const START = TimeStamp.seconds(1_750_000_000);
const SPAN = TimeSpan.minutes(10);
const FRAME = TimeSpan.milliseconds(16);
const GROUPS_PER_COLUMN = 1 / 2;
const DATA_TYPES = { x: DataType.TIMESTAMP, y: DataType.FLOAT32 };

const core = new SyntheticCore({
  rate: 1000,
  start: START,
  now: START.add(TimeSpan.hours(1)),
});
const client = telemTest.createClient(core);
afterAll(async () => await client.close());

const view = (): View => ({
  bounds: new TimeRange(core.now().sub(SPAN), core.now()).numericBounds,
  width: 1000,
});

const createSource = (props: TiledChannelDataProps): TiledChannelData =>
  new TiledChannelData(client, props, undefined, client.now, TimeSpan.ZERO);

const channel = core.data[0].key;
const x = createSource({ channel, useIndexOfChannel: true, timeSpan: SPAN });
const y = createSource({ channel, timeSpan: SPAN });

const read = (): DrawOperation[] => {
  const v = { view: view() };
  return buildDrawOperations(x.value(v)[1], y.value(v)[1], DEFAULT_OVERLAP_THRESHOLD);
};

const deadline = TimeStamp.now().add(TimeSpan.seconds(30));
while (x.loading() || y.loading() || x.fetching() || y.fetching()) {
  if (TimeStamp.now().after(deadline)) throw new Error("sources did not settle");
  read();
  await sleep.sleep(TimeSpan.milliseconds(1));
}

let ops = read();
const size = groupSize(ops, view(), GROUPS_PER_COLUMN);

const cold = (aggregation: Aggregation) => (): void =>
  void new Reducer().reduce(ops, aggregation, size, DATA_TYPES);

const warm = new Reducer();

test("reduce a 10 minute live window of 1 kHz samples", async ({ bench }) => {
  await bench.compare(
    bench("group size", () => void groupSize(ops, view(), GROUPS_PER_COLUMN)),
    bench("min_max, no reduced pairs held", cold("min_max")),
    bench("average, no reduced pairs held", cold("average")),
    bench("decimate, no reduced pairs held", cold("decimate")),
    bench(
      "min_max, every reduced pair held",
      { beforeAll: () => void warm.reduce(ops, "min_max", size, DATA_TYPES) },
      () => void warm.reduce(ops, "min_max", size, DATA_TYPES),
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
      () => void warm.reduce(ops, "min_max", size, DATA_TYPES),
    ),
  );
});
