// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeRange, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { channel } from "@/channel";
import { alignmentOf, RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { Joins, type Wait } from "@/framer/cache/join";
import { Store } from "@/framer/cache/store";
import { ValidationError } from "@/index";

const INDEX = 1;
const DATA = 2;
const BLOCK = 100;
const RANGE = tileRange({ level: RAW_LEVEL, index: BLOCK });

const at = (seconds: number): TimeStamp => RANGE.start.add(TimeSpan.seconds(seconds));

const spanOf = (seconds: number[]): TimeRange =>
  new TimeRange(at(seconds[0]), at(seconds[seconds.length - 1]).add(1));

const createIndex = (seconds: number[], alignment: bigint): Series =>
  new Series({
    data: new BigInt64Array(seconds.map((s) => at(s).valueOf())),
    dataType: DataType.TIMESTAMP,
    alignment,
    timeRange: spanOf(seconds),
    clock: INDEX,
  });

const createData = (
  values: number[],
  alignment: bigint,
  timeRange: TimeRange = TimeRange.ZERO,
): Series =>
  // The Core stamps each streamed series with the clock of its channel.
  new Series({ data: new Float32Array(values), alignment, timeRange, clock: INDEX });

const createFrame = (...entries: Array<[channel.Key, Series]>) =>
  new Map(entries.map(([k, s]) => [k, [s]]));

const createJoins = () => {
  const store = new Store({ dynamicBufferSize: 100 });
  const joins = new Joins(store);
  joins.join(DATA, INDEX);
  return { store, joins };
};

// Lands an empty catch-up read for both members and writes one placed frame.
const createJoined = () => {
  const { store, joins } = createJoins();
  joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
  joins.write(
    createFrame([INDEX, createIndex([1, 2], 0n)], [DATA, createData([10, 20], 0n)]),
  );
  return { store, joins };
};

// Returns the waits of the given members, as a catch-up read of them begins.
const waitsOf = (joins: Joins, ...keys: channel.Key[]): Wait[] =>
  joins.waiting(INDEX).filter((w) => keys.includes(w.key));

const keysOf = (joins: Joins): channel.Key[] => joins.waiting(INDEX).map((w) => w.key);

const leadingOf = (store: Store, ch: channel.Key): Series | null =>
  store.open(ch)?.buffer?.leadingBuffer ?? null;

const streamed = (store: Store, ch: channel.Key): number[] =>
  Array.from(leadingOf(store, ch) ?? []).map(Number);

// Returns the series of the channel's raw blocks in time order, the open one last.
const seriesOf = (store: Store, ch: channel.Key, ...blocks: number[]): Series[] => {
  const fetched = blocks.flatMap((b) => [...store.raw(ch, b).fetched]);
  const leading = leadingOf(store, ch);
  return leading == null ? fetched : [...fetched, leading];
};

// Returns whether each series after the first starts where the one before it ends.
const touches = (series: Series[]): boolean[] =>
  series.slice(1).map((s, i) => series[i].timeRange.end.equals(s.timeRange.start));

// Fills the block with a read of both members' samples at the seconds.
const fillRead = (store: Store, seconds: number[]): void => {
  const span = spanOf(seconds);
  const alignment = alignmentOf(BLOCK, 0);
  store
    .raw(INDEX, BLOCK)
    .fill([createIndex(seconds, alignment)], span.end, seconds.length);
  store
    .raw(DATA, BLOCK)
    .fill([createData(seconds, alignment, span)], span.end, seconds.length);
};

const NEXT_DOMAIN = alignmentOf(1, 0);


describe("scratch clocks", () => {
  it("should keep the index clock on every buffered and flushed series", () => {
    const { store, joins } = createJoined();
    let a = 2n;
    let t = 3;
    for (let i = 0; i < 60; i++) {
      const n = 1 + (i % 7);
      const secs = Array.from({ length: n }, (_, j) => t + j * 0.01);
      t += n * 0.01 + 0.001;
      joins.write(
        createFrame(
          [INDEX, createIndex(secs, a)],
          [DATA, createData(secs.map((s) => s * 10), a)],
        ),
      );
      a += BigInt(n);
    }
    for (const ch of [INDEX, DATA])
      for (const s of seriesOf(store, ch, BLOCK)) expect(s.clock).toBe(INDEX);
  });

  it("should keep the clock of data held before its index", () => {
    const { store, joins } = createJoined();
    joins.write(createFrame([DATA, createData([30, 40], 2n)]));
    joins.write(createFrame([INDEX, createIndex([3, 4], 2n)]));
    joins.write(
      createFrame([INDEX, createIndex([5], 4n)], [DATA, createData([50], 4n)]),
    );
    for (const ch of [INDEX, DATA])
      for (const s of seriesOf(store, ch, BLOCK)) expect(s.clock).toBe(INDEX);
    for (const s of joins.held(DATA)) expect(s.clock).toBe(INDEX);
  });
});
