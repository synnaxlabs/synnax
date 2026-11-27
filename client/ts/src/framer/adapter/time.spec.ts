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
import { type ChannelRetriever, ReadAdapter } from "@/framer/adapter";
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

const retrieveChannels: ChannelRetriever = async (params) => {
  const wanted = (Array.isArray(params) ? params : [params]) as Array<
    channel.Key | channel.Name
  >;
  return [INDEX, DATA, CALC].filter(
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
