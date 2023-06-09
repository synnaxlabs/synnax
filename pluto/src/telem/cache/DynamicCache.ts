// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, LazyArray, TimeRange, TimeStamp } from "@synnaxlabs/x";

/**
 * A cache for channel data that maintains a single, rolling LazyArray as a buffer
 * for channel data.
 */
export class DynamicCache {
  buffer: LazyArray;
  private readonly dataType: DataType;
  private readonly cap: number;

  /**
   * @constructor
   *
   * @param cap - The capacity of the cache buffer.
   * @param dataType - The data type of the channel.
   */
  constructor(cap: number, dataType: DataType) {
    this.buffer = this.allocate(cap);
    this.cap = cap;
  }

  /** @returns the number of samples currenly held in the cache. */
  get length(): number {
    return this.buffer.length;
  }

  /**
   * Writes the given arrays to the cache.
   *
   * @returns a list of buffers that were filled by the cache during the write. If
   * the current buffer is able to fit all writes, no buffers will be returned.
   */
  write(arrays: LazyArray[]): LazyArray[] {
    return arrays.flatMap((arr) => this._write(arr));
  }

  /**
   * Performs a 'dirty' read on the cache i.e. returns the entire buffer if its time
   * range overlaps AT ALL with the given time range.
   *
   * @param tr - The time range to read.
   * @returns the buffer if it overlaps with the given time range, null otherwise.
   */
  dirtyRead(tr: TimeRange): LazyArray | null {
    if (this.buffer.timeRange.overlapsWith(tr) && this.buffer.length > 0)
      return this.buffer;
    return null;
  }

  private allocate(length: number): LazyArray {
    return LazyArray.alloc(length, DataType.FLOAT32, TimeStamp.now().spanRange(0));
  }

  private _write(arr: LazyArray): LazyArray[] {
    const amountWritten = this.buffer.write(arr);
    if (amountWritten === arr.length) return [];
    const out = this.buffer;
    this.buffer = this.allocate(this.cap);
    return [out, ...this._write(arr.slice(amountWritten))];
  }
}
