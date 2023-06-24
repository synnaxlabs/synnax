// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeRange, TimeStamp } from "@synnaxlabs/x";

import { convertSeriesFloat32 } from "@/telem/convertSeries";

/**
 * A cache for channel data that maintains a single, rolling Series as a buffer
 * for channel data.
 */
export class DynamicCache {
  buffer: Series;
  private readonly cap: number;
  private readonly dataType: DataType;

  /**
   * @constructor
   *
   * @param cap - The capacity of the cache buffer.
   * @param dataType - The data type of the channel.
   */
  constructor(cap: number, dataType: DataType) {
    this.cap = cap;
    this.dataType = dataType;
    this.buffer = this.allocate(cap);
  }

  /** @returns the number of samples currenly held in the cache. */
  get length(): number {
    return this.buffer?.length ?? 0;
  }

  /**
   * Writes the given arrays to the cache.
   *
   * @returns a list of buffers that were filled by the cache during the write. If
   * the current buffer is able to fit all writes, no buffers will be returned.
   */
  write(series: Series[]): Series[] {
    return series.flatMap((arr) => this._write(arr));
  }

  /**
   * Performs a 'dirty' read on the cache i.e. returns the entire buffer if its time
   * range overlaps AT ALL with the given time range.
   *
   * @param tr - The time range to read.
   * @returns the buffer if it overlaps with the given time range, null otherwise.
   */
  dirtyRead(tr: TimeRange): Series | null {
    // if (!this.buffer.timeRange.overlapsWith(tr)) return null;
    // return this.buffer;
    return null;
  }

  private allocate(length: number): Series {
    const start = TimeStamp.now();
    return Series.alloc(
      length,
      DataType.FLOAT32,
      start.spanRange(TimeStamp.MAX),
      this.dataType.equals(DataType.TIMESTAMP) ? -start.valueOf() : 0
    );
  }

  private _write(series: Series): Series[] {
    const converted = convertSeriesFloat32(series, this.buffer.sampleOffset);
    const amountWritten = this.buffer.write(converted);
    if (amountWritten === series.length) return [];
    const out = this.buffer;
    this.buffer = this.allocate(this.cap);
    return [out, ...this._write(series.slice(amountWritten))];
  }
}
