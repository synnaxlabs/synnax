// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Stamps } from "@/framer/cache/stamps";

const T0 = TimeStamp.seconds(1_700_000_000);
const T1 = T0.add(TimeSpan.seconds(1));
const T2 = T0.add(TimeSpan.seconds(2));
const T3 = T0.add(TimeSpan.seconds(3));
const T4 = T0.add(TimeSpan.seconds(4));
const LEADING = (BigInt(0xffffffff) - 1_000_000n) << 32n;

const write = (alignment: bigint, length: number, stamp: TimeStamp): Series =>
  new Series({
    data: new Float32Array(length),
    dataType: DataType.FLOAT32,
    timeRange: stamp.range(stamp.add(TimeSpan.NANOSECOND)),
    alignment,
  });

describe("Stamps", () => {
  it("should return the stamp of the write holding the sample", () => {
    const s = new Stamps(4);
    s.record(write(10n, 3, T0));
    s.record(write(13n, 2, T1));
    expect(s.at(10n)).toEqual(T0);
    expect(s.at(12n)).toEqual(T0);
    expect(s.at(13n)).toEqual(T1);
    expect(s.at(14n)).toEqual(T1);
  });

  it("should be null before the first write and past the last", () => {
    const s = new Stamps(4);
    s.record(write(10n, 3, T0));
    expect(s.at(9n)).toBeNull();
    expect(s.at(13n)).toBeNull();
  });

  it("should be null between two writes", () => {
    const s = new Stamps(4);
    s.record(write(10n, 2, T0));
    s.record(write(20n, 1, T1));
    expect(s.at(15n)).toBeNull();
  });

  it("should forget the oldest write at capacity", () => {
    const s = new Stamps(2);
    s.record(write(0n, 1, T0));
    s.record(write(1n, 1, T1));
    s.record(write(2n, 1, T2));
    expect(s.at(0n)).toBeNull();
    expect(s.at(1n)).toEqual(T1);
    expect(s.at(2n)).toEqual(T2);
  });

  it("should find every kept write after the ring wraps", () => {
    const s = new Stamps(3);
    [T0, T1, T2, T3, T4].forEach((t, i) => s.record(write(BigInt(i) * 2n, 2, t)));
    expect(s.at(3n)).toBeNull();
    expect(s.at(4n)).toEqual(T2);
    expect(s.at(7n)).toEqual(T3);
    expect(s.at(9n)).toEqual(T4);
    expect(s.at(10n)).toBeNull();
  });

  it("should replace the writes from a repeated alignment on", () => {
    const s = new Stamps(4);
    s.record(write(0n, 3, T0));
    s.record(write(3n, 2, T1));
    s.record(write(3n, 4, T2));
    expect(s.at(1n)).toEqual(T0);
    expect(s.at(3n)).toEqual(T2);
    expect(s.at(6n)).toEqual(T2);
    expect(s.at(7n)).toBeNull();
  });

  it("should forget every write when the alignment regresses", () => {
    const s = new Stamps(4);
    s.record(write(100n, 2, T0));
    s.record(write(102n, 2, T1));
    s.record(write(0n, 1, T2));
    expect(s.at(100n)).toBeNull();
    expect(s.at(102n)).toBeNull();
    expect(s.at(0n)).toEqual(T2);
  });

  it("should hold a leading alignment above the signed 64-bit range", () => {
    const s = new Stamps(4);
    s.record(write(LEADING, 2, T0));
    s.record(write(LEADING + 2n, 1, T1));
    expect(s.at(LEADING - 1n)).toBeNull();
    expect(s.at(LEADING + 1n)).toEqual(T0);
    expect(s.at(LEADING + 2n)).toEqual(T1);
    expect(s.at(LEADING + 3n)).toBeNull();
  });

  it("should record a fetched write past the last record", () => {
    const s = new Stamps(4);
    s.record(write(10n, 2, T0));
    s.recordFetched(write(12n, 1, T1));
    expect(s.at(11n)).toEqual(T0);
    expect(s.at(12n)).toEqual(T1);
  });

  it("should keep live records when a fetched write lands behind them", () => {
    const s = new Stamps(4);
    s.record(write(10n, 3, T1));
    s.recordFetched(write(5n, 1, T0));
    s.recordFetched(write(12n, 1, T2));
    expect(s.at(5n)).toBeNull();
    expect(s.at(10n)).toEqual(T1);
    expect(s.at(12n)).toEqual(T1);
  });

  it("should record nothing for an empty or unstamped series", () => {
    const s = new Stamps(4);
    s.record(write(0n, 0, T0));
    s.record(
      new Series({
        data: new Float32Array(2),
        dataType: DataType.FLOAT32,
        alignment: 5n,
      }),
    );
    expect(s.at(0n)).toBeNull();
    expect(s.at(5n)).toBeNull();
  });
});
