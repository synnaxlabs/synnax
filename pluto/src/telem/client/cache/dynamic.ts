// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeStamp } from "@synnaxlabs/x";

import { convertSeriesFloat32 } from "@/telem/aether/convertSeries";

export interface DynamicWriteResponse {
  flushed: Series[];
  allocated: Series[];
}

/**
 * A cache for channel data that maintains a single, rolling Series as a buffer
 * for channel data.
 */
export class Dynamic {
  buffer: Series | null;
  private readonly cap: number;
  private readonly dataType: DataType;
  private counter = 0;

  /**
   * @constructor
   *
   * @param cap - The capacity of the cache buffer.
   * @param dataType - The data type of the channel.
   */
  constructor(cap: number, dataType: DataType) {
    this.cap = cap;
    this.dataType = dataType;
    this.buffer = null;
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
  write(series: Series[]): DynamicWriteResponse {
    const responses = series.flatMap((arr) => this._write(arr));
    return {
      flushed: responses.flatMap((res) => res.flushed),
      allocated: responses.flatMap((res) => res.allocated),
    };
  }

  private allocate(length: number, alignment: number, start: TimeStamp): Series {
    this.counter++;
    return Series.alloc({
      length,
      dataType: DataType.FLOAT32,
      timeRange: start.spanRange(TimeStamp.MAX),
      sampleOffset: this.dataType.equals(DataType.TIMESTAMP)
        ? BigInt(start.valueOf())
        : 0,
      glBufferUsage: "dynamic",
      alignment,
      key: `dynamic-${this.counter}`,
    });
  }

  private _write(series: Series): DynamicWriteResponse {
    const res: DynamicWriteResponse = { flushed: [], allocated: [] };
    // This only happens on the first write to the cache
    if (this.buffer == null) {
      this.buffer = this.allocate(this.cap, series.alignment, TimeStamp.now());
      res.allocated.push(this.buffer);
    } else if (
      Math.abs(this.buffer.alignment + this.buffer.length - series.alignment) > 1
    ) {
      // This case occurs when the alignment of the incoming series does not match
      // the alignment of the current buffer. In this case, we flush the current buffer
      // and allocate a new one.
      const now = TimeStamp.now();
      this.buffer.timeRange.end = now;
      res.flushed.push(this.buffer);
      this.buffer = this.allocate(this.cap, series.alignment, now);
      res.allocated.push(this.buffer);
    }
    const converted = convertSeriesFloat32(series, this.buffer.sampleOffset);
    const amountWritten = this.buffer.write(converted);
    // This means that the current buffer is large enough to fit the entire incoming
    // series. We're done in this case.
    if (amountWritten === series.length) return res;
    // Push the current buffer to the flushed list.
    const now = TimeStamp.now();
    this.buffer.timeRange.end = now;
    res.flushed.push(this.buffer);
    this.buffer = this.allocate(this.cap, series.alignment + amountWritten, now);
    res.allocated.push(this.buffer);
    const nextRes = this._write(series.slice(amountWritten));
    res.flushed.push(...nextRes.flushed);
    res.allocated.push(...nextRes.allocated);
    return res;
  }
}
