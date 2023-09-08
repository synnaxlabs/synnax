// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeStamp } from "@synnaxlabs/x";

import { convertSeriesFloat32 } from "@/telem/core/convertSeries";

/**
 * A cache for channel data that maintains a single, rolling Series as a buffer
 * for channel data.
 */
export class Dynamic {
  buffer: Series | null;
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
  write(series: Series[]): Series[] {
    return series.flatMap((arr) => this._write(arr));
  }

  private allocate(length: number, alignment: number): Series {
    const start = TimeStamp.now();
    return Series.alloc(
      length,
      DataType.FLOAT32,
      start.spanRange(TimeStamp.MAX),
      this.dataType.equals(DataType.TIMESTAMP) ? start.valueOf() : 0,
      "static",
      alignment,
    );
  }

  private _write(series: Series): Series[] {
    if (this.buffer == null) this.buffer = this.allocate(this.cap, series.alignment);
    const converted = convertSeriesFloat32(series, this.buffer.sampleOffset);
    const amountWritten = this.buffer.write(converted);
    if (amountWritten === series.length) return [];
    const out = this.buffer;
    this.buffer = this.allocate(this.cap, series.alignment);
    return [out, ...this._write(series.slice(amountWritten))];
  }
}
