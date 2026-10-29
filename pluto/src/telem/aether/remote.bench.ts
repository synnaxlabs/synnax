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
  dynamicWindow,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { allocSuite } from "@synnaxlabs/x/bench";
import { afterAll, test } from "vitest";

import { ChannelData, type ChannelDataProps, type Client } from "@/telem/aether/remote";
import { type ValueProps } from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/testutil";

// Each read is one frame of a line 1000 pixels wide, with every tile it draws cached.

const START = TimeStamp.seconds(1_750_000_000);
const NOW = START.add(TimeSpan.days(7));
const WEEK = new TimeRange(START, NOW);
const LIVE_SPAN = TimeSpan.minutes(10);
const FRAME = TimeSpan.milliseconds(16);
const WIDTH = 1000;

const closers: (() => Promise<void>)[] = [];
afterAll(async () => await Promise.all(closers.map(async (close) => await close())));

const createClient = (): { core: SyntheticCore; client: Client } => {
  const core = new SyntheticCore({ rate: 1000, start: START, now: NOW });
  const client = telemTest.createClient(core);
  closers.push(async () => await client.feed.close());
  return { core, client };
};

const viewOf = (tr: TimeRange): ValueProps => ({
  bounds: tr.numericBounds,
  points: WIDTH,
});

const liveView = (core: SyntheticCore): ValueProps =>
  viewOf(new TimeRange(core.now().sub(LIVE_SPAN), core.now()));

const createSource = async (
  client: Client,
  props: ChannelDataProps,
  view: ValueProps,
): Promise<ChannelData> => {
  const source = new ChannelData(client, props, client.now);
  await settle(source, view);
  return source;
};

// settle reads the view until the source holds everything the view draws.
const settle = async (source: ChannelData, view: ValueProps): Promise<void> => {
  const deadline = TimeStamp.now().add(TimeSpan.seconds(30));
  while (TimeStamp.now().before(deadline)) {
    if (source.value(view).status.variant !== "loading") return;
    await sleep.sleep(TimeSpan.milliseconds(1));
  }
  throw new Error("source did not settle");
};

// yieldTask lets the tile reads and streamed frames that arrive between frames land.
const yieldTask = async (): Promise<void> =>
  await new Promise((resolve) => setImmediate(resolve));

const staticRead = async (view: ValueProps): Promise<() => void> => {
  const { core, client } = createClient();
  const source = await createSource(
    client,
    { channel: core.data[0].key, window: staticWindow(WEEK) },
    view,
  );
  return () => void source.value(view);
};

const liveRead = async (): Promise<{
  core: SyntheticCore;
  source: ChannelData;
}> => {
  const { core, client } = createClient();
  const source = await createSource(
    client,
    { channel: core.data[0].key, window: dynamicWindow(LIVE_SPAN) },
    liveView(core),
  );
  return { core, source };
};

const HOUR = new TimeRange(
  START.add(TimeSpan.days(3)),
  START.add(TimeSpan.days(3)).add(TimeSpan.hours(1)),
);

const homeRead = await staticRead(viewOf(WEEK));
const hourRead = await staticRead(viewOf(HOUR));
const live = await liveRead();
const liveHomeRead = (): void => void live.source.value(liveView(live.core));

test("read a held view", async ({ bench }) => {
  await bench.compare(
    bench("static week, week view", homeRead),
    bench("static week, hour view", hourRead),
    bench("live 10 minutes at 1 kHz, no new samples", liveHomeRead),
  );
});

// A pan moves the view by a sixtieth of its span per frame. It sweeps one day back and
// forth, so after the first sweep every tile it crosses comes from the cache.
const pan = async (): Promise<() => Promise<void>> => {
  const { core, client } = createClient();
  const span = HOUR.span.valueOf();
  const step = span / 60n;
  const steps = Number(TimeSpan.days(1).valueOf() / step);
  let i = 0;
  const next = (): ValueProps => {
    const phase = i++ % (2 * steps);
    const offset = BigInt(phase < steps ? phase : 2 * steps - phase) * step;
    const start = HOUR.start.add(offset);
    return viewOf(new TimeRange(start, start.add(span)));
  };
  const source = await createSource(
    client,
    { channel: core.data[0].key, window: staticWindow(WEEK) },
    next(),
  );
  for (let s = 0; s < 2 * steps; s++) {
    source.value(next());
    await yieldTask();
  }
  await settle(source, next());
  return async () => {
    source.value(next());
    await yieldTask();
  };
};

const stream = async (): Promise<() => Promise<void>> => {
  const { core, source } = await liveRead();
  return async () => {
    core.advance(FRAME);
    await yieldTask();
    source.value(liveView(core));
  };
};

// Each async read includes one yield, which the baseline measures alone.
test("read while data moves", async ({ bench }) => {
  await bench.compare(
    bench("baseline: one yield", yieldTask),
    bench("pan an hour view", await pan()),
    bench("live 10 minutes at 1 kHz, 16 ms of new samples", await stream()),
  );
});

allocSuite("held view read", [
  ["static week, week view", homeRead],
  ["static week, hour view", hourRead],
  ["live 10 minutes at 1 kHz, no new samples", liveHomeRead],
]);
