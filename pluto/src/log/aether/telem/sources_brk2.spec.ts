// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  type calculation,
  channel,
  DataType,
  framer,
  query,
  type Synnax as Client,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  SyntheticCore,
} from "@synnaxlabs/client/testutil";
import { id, Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";

import { StreamMultiChannelLog } from "@/log/aether/telem/sources";
import { type telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";

const POLL = { timeout: 10_000, interval: 20 };

describe("StreamMultiChannelLog with a Core (brk2)", () => {
  const admin = createTestClient();
  const closers: Array<() => Promise<void> | void> = [];
  afterEach(async () => {
    for (const close of closers.splice(0).reverse()) await close();
  });

  const connect = (c: Client): telem.Client => {
    const feed = c.openFeed();
    closers.push(async () => await feed.close());
    return {
      feed,
      channels: c.channels,
      calculations: c.calculations,
      now: () => TimeStamp.now(),
    };
  };

  const open = (client: telem.Client, keys: channel.Key[]): StreamMultiChannelLog => {
    const log = new StreamMultiChannelLog(client, { channels: keys });
    closers.push(() => log.cleanup());
    log.value();
    return log;
  };

  // A calculation over a stored channel, with samples written ten seconds ago.
  const createCalculated = async () => {
    const time = await admin.channels.create({
      name: id.create(),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await admin.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT64,
      index: time.key,
    });
    const calc = await admin.calculations.create({
      name: id.create(),
      expression: `return ${data.name} * 2`,
    });
    const output = await admin.channels.retrieve(calc.output);
    const start = TimeStamp.now().sub(TimeSpan.seconds(10));
    const stamps = [0, 1, 2].map((i) => start.add(TimeSpan.milliseconds(i)));
    await admin.write(start, { [time.key]: stamps, [data.key]: [1, 2, 3] });
    return { time, data, output, start };
  };

  it("should know the stored history of a calculated channel for a reader that cannot read its calculation", async () => {
    const { time, data, output, start } = await createCalculated();
    expect(output.virtual).toBe(true);
    expect(output.index).not.toBe(0);
    const reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [
        channel.resourceID(time.key),
        channel.resourceID(data.key),
        channel.resourceID(output.key),
        channel.resourceID(output.index),
        framer.TYPE_RESOURCE_ID,
        user.TYPE_RESOURCE_ID,
        access.role.TYPE_RESOURCE_ID,
        access.policy.TYPE_RESOURCE_ID,
      ],
      actions: ["retrieve"],
    });
    const asAdmin = connect(admin);
    const asReader = connect(reader);
    // The reader can read the channel and its stored history, but not its calculation.
    expect((await reader.channels.retrieve(output.key)).key).toBe(output.key);
    expect(await reader.calculations.outputs.retrieve([output.key])).toEqual([
      { key: output.key, hidden: true },
    ]);
    expect((await asReader.feed.firstTime([output.key]))?.valueOf()).toBe(
      start.valueOf(),
    );
    const adminLog = open(asAdmin, [output.key]);
    await expect
      .poll(() => adminLog.timeRange?.start.valueOf(), POLL)
      .toBe(start.valueOf());
    const readerLog = open(asReader, [output.key]);
    await expect
      .poll(() => readerLog.timeRange?.start.valueOf(), POLL)
      .toBe(start.valueOf());
  });

  it("should show a channel after the cache resets during its output lookup", async () => {
    const start = TimeStamp.seconds(1_000_000);
    const core = new SyntheticCore({
      rate: Rate.hz(10),
      start,
      now: start.add(TimeSpan.seconds(30)),
      channels: 1,
    });
    const [a] = core.data.map((c) => c.key);
    const cache = new query.Cache({ openStreamer: null, onError: () => {} });
    const { promise: answered, resolve: answer } = Promise.withResolvers<void>();
    let lookups = 0;
    const outputs = cache.createTable<channel.Key, calculation.Output>({
      name: "calculation.outputs",
      fetch: async (keys) => {
        if (++lookups === 1) await answered;
        return keys.map((key) => ({ key, calculation: null }));
      },
    });
    const base = telemTest.createClient(core, {
      batchDebounce: TimeSpan.milliseconds(5),
      removalDelay: TimeSpan.ZERO,
    });
    closers.push(async () => await base.feed.close());
    const posted: string[] = [];
    const log = new StreamMultiChannelLog(
      { ...base, calculations: { outputs } },
      { channels: [a] },
      {
        onStatusChange: (s) => {
          if (s.variant === "error") posted.push(`${s.message}: ${s.description}`);
        },
      },
      core.now,
      undefined,
      { baseInterval: TimeSpan.milliseconds(10) },
    );
    closers.push(() => log.cleanup());
    log.value();
    await expect.poll(() => lookups, POLL).toBe(1);
    // The Core behind the connection is replaced while the lookup is in flight.
    await cache.reset();
    answer();
    await expect.poll(() => log.value().data.length, POLL).toBe(300);
    expect(posted).toEqual([]);
  });
});
