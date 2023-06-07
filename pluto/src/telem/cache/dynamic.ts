// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, LazyArray, TimeRange, TimeStamp } from "@synnaxlabs/x";

export class DynamicCache {
  private readonly dataType: DataType;
  curr: LazyArray;
  private readonly cap: number;

  constructor(cap: number, dataType: DataType) {
    this.dataType = dataType;
    this.curr = this.allocate(cap);
    this.cap = cap;
  }

  get length(): number {
    return this.curr.length;
  }

  get data(): LazyArray {
    return this.curr;
  }

  write(arrays: LazyArray[]): LazyArray[] {
    return arrays.flatMap((arr) => this._write(arr));
  }

  read(tr: TimeRange): LazyArray | null {
    if (this.curr.timeRange.overlapsWith(tr)) return this.curr;
    return null;
  }

  private allocate(length: number): LazyArray {
    return LazyArray.alloc(length, this.dataType, TimeStamp.now().spanRange(0));
  }

  private _write(arr: LazyArray): LazyArray[] {
    const amountWritten = this.curr.write(arr);
    if (amountWritten === arr.length) return [];
    const next = this.allocate(this.cap);
    return [next, ...this._write(arr.slice(amountWritten))];
  }
}
