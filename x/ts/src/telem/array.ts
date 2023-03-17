// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  convertDataType,
  DataType,
  NativeTypedArray,
  Size,
  TimeRange,
  UnparsedDataType,
} from "./telem";

import { Compare } from "@/compare";

export type SampleValue = number | bigint;

const validateFieldNotNull = (name: string, field: unknown): void => {
  if (field == null) {
    throw new Error(`field ${name} is null`);
  }
};

/**
 * A strongly typed array of telemetry samples backed
 * by an underlying binary buffer
 */
export class LazyArray {
  readonly dataType: DataType;
  offset: SampleValue;
  private readonly _data: ArrayBufferLike;
  readonly _timeRange?: TimeRange;
  private _min?: SampleValue;
  private _max?: SampleValue;

  constructor(
    data: ArrayBufferLike | NativeTypedArray,
    dataType?: UnparsedDataType,
    timeRange?: TimeRange,
    offset?: SampleValue
  ) {
    if (
      dataType == null &&
      !(data instanceof ArrayBuffer) &&
      !(data instanceof SharedArrayBuffer)
    ) {
      this.dataType = new DataType(data);
    } else if (dataType != null) {
      this.dataType = new DataType(dataType);
    } else {
      throw new Error(
        "must provide a data type when constructing a LazyArray from a buffer"
      );
    }
    this.offset = offset ?? 0;
    this._data = data;
    this._timeRange = timeRange;
  }

  /** @returns the underlying buffer backing this array. */
  get buffer(): ArrayBufferLike {
    return this._data;
  }

  /** @returns a native typed array with the proper data type. */
  get data(): NativeTypedArray {
    validateFieldNotNull("dataType", this._data);
    return new this.dataType.Array(this._data);
  }

  /** @returns the time range of this array. */
  get timeRange(): TimeRange {
    validateFieldNotNull("_timeRange", this._timeRange);
    return this._timeRange as TimeRange;
  }

  /** @returns the size of the underlying buffer in bytes. */
  get size(): Size {
    return new Size(this.buffer.byteLength);
  }

  /** @returns the number of samples in this array. */
  get length(): number {
    return this.dataType.density.length(this.size);
  }

  /**
   * Creates a new array with a different data type.
   * @param target the data type to convert to.
   * @param offset an offset to apply to each sample. This can help with precision
   * issues when converting between data types.
   *
   * WARNING: This method is expensive and copies the entire underlying array. There
   * also may be untimely precision issues when converting between data types.
   */
  convert(target: DataType, offset: SampleValue = 0): LazyArray {
    if (this.dataType.equals(target)) return this;
    const data = new target.Array(this.length);
    for (let i = 0; i < this.length; i++) {
      data[i] = convertDataType(this.dataType, target, this.data[i], offset);
    }
    const n = new LazyArray(data.buffer, target, this._timeRange, offset);
    if (this._max != null) n._max = addSamples(this._max, offset);
    if (this._min != null) n._min = addSamples(this._min, offset);
    return n;
  }

  get max(): number | bigint {
    if (this._max == null) {
      if (this.dataType.equals(DataType.TIMESTAMP)) {
        this._max = this.data[this.data.length - 1];
      } else if (this.dataType.usesBigInt) {
        const d = this.data as BigInt64Array;
        this._max = d.reduce((a, b) => (a > b ? a : b));
      } else {
        const d = this.data as Float64Array;
        this._max = d.reduce((a, b) => (a > b ? a : b));
      }
    }
    return addSamples(this._max, this.offset);
  }

  get min(): number | bigint {
    if (this._min == null) {
      if (this.dataType.equals(DataType.TIMESTAMP)) {
        this._min = this.data[0];
      } else if (this.dataType.usesBigInt) {
        const d = this.data as BigInt64Array;
        this._min = d.reduce((a, b) => (a < b ? a : b));
      } else {
        const d = this.data as Float64Array;
        this._min = d.reduce((a, b) => (a < b ? a : b));
      }
    }
    return addSamples(this._min, this.offset);
  }

  enrich(): void {
    let _ = this.max;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    _ = this.min;
  }

  get range(): number | bigint {
    return addSamples(this.max, -this.min);
  }

  binarySearch(value: SampleValue): number {
    let left = 0;
    let right = this.length - 1;
    const compare = Compare.newF(value);
    while (left <= right) {
      const mid = Math.floor((left + right) / 2);
      const cmp = compare(this.data[mid], value);
      if (cmp === 0) return mid;
      if (cmp < 0) left = mid + 1;
      else right = mid - 1;
    }
    return left;
  }
}

export const addSamples = (a: SampleValue, b: SampleValue): SampleValue => {
  if (typeof a === "bigint" && typeof b === "bigint") return a + b;
  else if (typeof a === "number" && typeof b === "number") return a + b;
  return Number(a) + Number(b);
};
