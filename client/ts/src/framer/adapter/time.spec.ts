// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { type ChannelRetriever, ReadAdapter, WriteAdapter } from "@/framer/adapter";
import { Frame, NotFoundError, ValidationError } from "@/index";

const INDEX = new channel.Channel({
  key: 1,
  name: "a_time",
  dataType: DataType.TIMESTAMP,
  isIndex: true,
}).payload;
const DATA = new channel.Channel({
  key: 3,
  name: "d",
  dataType: DataType.FLOAT64,
  index: 1,
}).payload;
const CALC = new channel.Channel({
  key: 4,
  name: "calc",
  dataType: DataType.FLOAT64,
  virtual: true,
  expression: "return d * 2",
}).payload;

// NAMED_TIME is a stored channel that takes the name time.
const NAMED_TIME = new channel.Channel({
  key: 5,
  name: "time",
  dataType: DataType.FLOAT64,
  virtual: true,
}).payload;

const retrieveChannels: ChannelRetriever = async (params) => {
  const wanted = (Array.isArray(params) ? params : [params]) as Array<
    channel.Key | channel.Name
  >;
  return [INDEX, DATA, CALC, NAMED_TIME].filter(
    (ch) => wanted.includes(ch.key) || wanted.includes(ch.name),
  );
};

const stamps = (values: bigint[], alignment: bigint, clock: number): Series =>
  new Series({
    data: new BigInt64Array(values),
    dataType: DataType.TIMESTAMP,
    alignment,
    clock,
  });

const values = (data: number[], alignment: bigint, clock: number): Series =>
  new Series({ data: new Float64Array(data), alignment, clock });

// read adapts frame as a read of keys.
const read = async (keys: channel.Key[] | channel.Name[], frame: Frame) =>
  (await ReadAdapter.open(retrieveChannels, keys)).adapt(frame);

describe("Frame.time across copies", () => {
  // timed returns a read of channel 3 whose time is on clock 1.
  const timed = async (): Promise<Frame> =>
    await read(
      [3],
      new Frame([3, 1], [values([1, 2], 0n, 1), stamps([5n, 6n], 0n, 1)]),
    );
  const times = (fr: Frame): bigint[] => Array.from(fr.time(3)) as bigint[];

  it("should keep the time of a frame that it pushes", async () => {
    const fr = new Frame();
    fr.push(await timed());
    expect(times(fr)).toEqual([5n, 6n]);
  });

  it("should keep the time in concat, filter, map, mapFilter, and get", async () => {
    const fr = await timed();
    expect(times(new Frame().concat(fr))).toEqual([5n, 6n]);
    expect(times(fr.filter(() => true))).toEqual([5n, 6n]);
    expect(times(fr.map((k, s) => [k, s]))).toEqual([5n, 6n]);
    expect(times(fr.mapFilter((k, s) => [k, s, true]))).toEqual([5n, 6n]);
    expect(times(fr.get([3]))).toEqual([5n, 6n]);
  });
});

describe("Frame.time", () => {
  it("should give a channel the time of its clock, which is not a channel", async () => {
    const fr = await read(
      [3],
      new Frame([3, 1], [values([1, 2], 0n, 1), stamps([5n, 6n], 0n, 1)]),
    );
    expect(fr.columns).toEqual([3]);
    expect(Array.from(fr.time(3))).toEqual([5n, 6n]);
  });

  it("should give a series the time series under its clock", async () => {
    const fr = await read(
      [3],
      new Frame([3, 1], [values([1], 0n, 1), stamps([5n], 0n, 1)]),
    );
    const time = fr.time(fr.get(3).series[0]);
    expect(Array.from(time)).toEqual([5n]);
    expect(time.alignment).toEqual(0n);
  });

  it("should give a series only the stamps at its own alignments", async () => {
    const fr = await read(
      [3],
      new Frame([3, 1], [values([2, 3], 1n, 1), stamps([5n, 6n, 7n, 8n], 0n, 1)]),
    );
    expect(Array.from(fr.time(3))).toEqual([6n, 7n]);
  });

  it("should give one time series per data series, with the same lengths", async () => {
    const fr = await read(
      [3],
      new Frame(
        [3, 3, 1, 1],
        [
          values([1, 2], 0n, 1),
          values([3], 10n, 1),
          stamps([5n, 6n], 0n, 1),
          stamps([9n, 10n], 10n, 1),
        ],
      ),
    );
    expect(fr.time(3).series.map((s) => s.length)).toEqual([2, 1]);
    expect(Array.from(fr.time(3))).toEqual([5n, 6n, 9n]);
  });

  it("should step through the alignments of a reduced read by its multiple", async () => {
    const data = new Series({
      data: new Float64Array([1, 2]),
      alignment: 4n,
      alignmentMultiple: 2n,
      clock: 1,
    });
    const time = new Series({
      data: new BigInt64Array([10n, 20n, 30n]),
      dataType: DataType.TIMESTAMP,
      alignment: 2n,
      alignmentMultiple: 2n,
      clock: 1,
    });
    const fr = await read([3], new Frame([3, 1], [data, time]));
    expect(Array.from(fr.time(3))).toEqual([20n, 30n]);
  });

  it("should fail for a channel whose time the frame does not hold", async () => {
    const fr = await read([3], new Frame([3], [values([1], 0n, 1)]));
    expect(() => fr.time(3)).toThrow(NotFoundError);
    expect(() => fr.time(3)).toThrow(
      "The frame holds no time for 3. The read marked it untimed",
    );
  });

  it("should hold a requested index as its channel and as time", async () => {
    const fr = await read(
      [1, 3],
      new Frame([1, 3], [stamps([5n, 6n], 0n, 1), values([1, 2], 0n, 1)]),
    );
    expect(Array.from(fr.get(1))).toEqual([5n, 6n]);
    expect(Array.from(fr.time(3))).toEqual([5n, 6n]);
    expect(Array.from(fr.time(1))).toEqual([5n, 6n]);
  });

  it("should keep the samples of an index that the read requests alone", async () => {
    const fr = await read([1], new Frame([1], [stamps([5n, 6n], 0n, 1)]));
    expect(Array.from(fr.get(1))).toEqual([5n, 6n]);
    expect(Array.from(fr.time(1))).toEqual([5n, 6n]);
  });

  it("should keep a calculation's stamps out of its index's entry", async () => {
    const fr = await read(
      [1, 4],
      new Frame([4, 1], [values([2], 0n, 1), stamps([5n], 0n, 1)]),
    );
    expect(fr.has(1)).toBe(false);
    expect(Array.from(fr.time(4))).toEqual([5n]);
  });

  it("should give the time of a channel read by name", async () => {
    const fr = await read(
      ["d"],
      new Frame([3, 1], [values([1], 0n, 1), stamps([5n], 0n, 1)]),
    );
    expect(Array.from(fr.time("d"))).toEqual([5n]);
  });

  it("should fail on time when no channel of the frame takes that name", async () => {
    const fr = await read(
      ["d"],
      new Frame([3, 1], [values([1], 0n, 1), stamps([5n], 0n, 1)]),
    );
    expect(() => fr.get("time")).toThrow(ValidationError);
    expect(() => fr.get("time")).toThrow(
      '"time" is not a channel. Writes take time under "time", and ' +
        "frame.time(channel) gives a channel's time",
    );
  });
});

describe("WriteAdapter time", () => {
  // open opens an adapter for channels whose writer lists the index of DATA as a clock.
  const open = async (channels: channel.Name[]): Promise<WriteAdapter> => {
    const adapter = await WriteAdapter.open(retrieveChannels, channels);
    adapter.clocks = [INDEX.key];
    return adapter;
  };
  const stamp = (): Series => stamps([5n], 0n, 0);

  it('should take "time" in every write form, beside a channel named time', async () => {
    const adapter = await open(["d", "time"]);
    const frames = [
      await adapter.adapt({ time: [5n], d: [1] }),
      await adapter.adapt(["time", "d"], [[5n], [1]]),
      await adapter.adapt(
        new Map<channel.Name, Series>([
          ["time", stamp()],
          ["d", values([1], 0n, 0)],
        ]),
      ),
    ];
    for (const fr of frames) expect(fr.columns).toEqual([DATA.key, INDEX.key]);
  });

  it("should write a channel named time by its key", async () => {
    const adapter = await open(["d", "time"]);
    expect((await adapter.adapt(NAMED_TIME.key, [1.5])).columns).toEqual([
      NAMED_TIME.key,
    ]);
  });

  it('should name the index channel that a write holds beside "time"', async () => {
    const adapter = await open(["d", "a_time"]);
    await expect(adapter.adapt({ time: [5n], d: [1], a_time: [5n] })).rejects.toThrow(
      'The write holds "time" and the index channel a_time, which both give the ' +
        "time of one group",
    );
  });

  it('should name both causes when a channel cannot take "time"', async () => {
    const adapter = await WriteAdapter.open(retrieveChannels, ["d"]);
    await expect(adapter.adapt({ time: [5n], d: [1] })).rejects.toThrow(
      'd cannot take "time": its index already stores stamps from the writer\'s ' +
        "start, or the writer may not write its index",
    );
  });
});
