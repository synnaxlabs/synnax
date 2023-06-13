// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Compare } from "@/compare";
import { Bounds } from "@/spatial/core";
import { GLBufferController, GLBufferUsage } from "@/telem/gl";
import {
  convertDataType,
  DataType,
  NativeTypedArray,
  Rate,
  Size,
  TimeRange,
  TimeStamp,
  UnparsedDataType,
} from "@/telem/telem";

export type SampleValue = number | bigint;

const validateFieldNotNull = (name: string, field: unknown): void => {
  if (field == null) {
    throw new Error(`field ${name} is null`);
  }
};

interface GL {
  control: GLBufferController | null;
  buffer: WebGLBuffer | null;
  prevBuffer: number;
  bufferUsage: GLBufferUsage;
}

const FULL_BUFFER = -1;

/**
 * A strongly typed array of telemetry samples backed
 * by an underlying binary buffer.
 */
export class LazyArray {
  /** The data type of the array */
  readonly dataType: DataType;
  /**
   * A sample offset that can be used to shift the values of all samples upwards or
   * downwards. Typically used to convert arrays to lower precision while preserving
   * the relative range of actual values.
   */
  sampleOffset: SampleValue;
  /**
   * Stores information about the buffer state of this array into a WebGL buffer.
   */
  private readonly gl: GL;
  /** The underlying data. */
  private readonly _data: ArrayBuffer;
  readonly _timeRange?: TimeRange;
  /** A cached minimum value. */
  private _min?: SampleValue;
  /** A cached maximum value. */
  private _max?: SampleValue;
  /** The write position of the buffer. */
  private pos: number = FULL_BUFFER;
  /** Tracks the number of entities currently using this array. */
  private refCount: number = 0;

  static alloc(
    length: number,
    dataType: UnparsedDataType,
    timeRange?: TimeRange
  ): LazyArray {
    if (length === 0)
      throw new Error("[LazyArray] - cannot allocate an array of length 0");
    const data = new new DataType(dataType).Array(length);
    const arr = new LazyArray(data.buffer, dataType, timeRange);
    arr.pos = 0;
    return arr;
  }

  static generateTimestamps(length: number, rate: Rate, start: TimeStamp): LazyArray {
    const tr = start.spanRange(rate.span(length));
    const data = new BigInt64Array(length);
    for (let i = 0; i < length; i++) {
      data[i] = BigInt(start.add(rate.span(i)).valueOf());
    }
    return new LazyArray(data, DataType.TIMESTAMP, tr);
  }

  constructor(
    data: ArrayBuffer | NativeTypedArray,
    dataType?: UnparsedDataType,
    timeRange?: TimeRange,
    sampleOffset?: SampleValue,
    glBufferUsage: GLBufferUsage = "static"
  ) {
    if (dataType == null && !(data instanceof ArrayBuffer)) {
      this.dataType = new DataType(data);
    } else if (dataType != null) {
      this.dataType = new DataType(dataType);
    } else {
      throw new Error(
        "must provide a data type when constructing a LazyArray from a buffer"
      );
    }
    this.sampleOffset = sampleOffset ?? 0;
    this._data = data;
    this._timeRange = timeRange;
    this.gl = {
      control: null,
      buffer: null,
      prevBuffer: 0,
      bufferUsage: glBufferUsage,
    };
  }

  acquire(gl?: GLBufferController): void {
    this.refCount++;
    if (gl != null) this.updateGLBuffer(gl);
  }

  release(gl?: GLBufferController): void {
    this.refCount--;
    if (this.refCount === 0 && gl != null) this.maybeGarbageCollectGLBuffer(gl);
    else if (this.refCount < 0)
      throw new Error("cannot release an array with a negative reference count");
  }

  write(other: LazyArray): number {
    if (!other.dataType.equals(this.dataType))
      throw new Error("buffer must be of the same type as this array");

    // We've filled the entire underlying buffer.
    if (this.pos === FULL_BUFFER) return 0;
    const available = this.cap - this.pos;

    const toWrite = available < other.length ? other.slice(0, available) : other;
    this.underlyingData.set(toWrite.data as any, this.pos);
    this.maybeRecomputeMinMax(toWrite);
    this.pos += toWrite.length;
    return toWrite.length;
  }

  /** @returns the underlying buffer backing this array. */
  get buffer(): ArrayBufferLike {
    return this._data;
  }

  private get underlyingData(): NativeTypedArray {
    return new this.dataType.Array(this._data);
  }

  /** @returns a native typed array with the proper data type. */
  get data(): NativeTypedArray {
    if (this.pos === FULL_BUFFER) return this.underlyingData;
    return new this.dataType.Array(this._data, 0, this.pos);
  }

  /** @returns the time range of this array. */
  get timeRange(): TimeRange {
    validateFieldNotNull("timeRange", this._timeRange);
    return this._timeRange as TimeRange;
  }

  /** @returns the capacity of the underlying buffer in bytes. */
  get byteCap(): Size {
    return new Size(this.buffer.byteLength);
  }

  /** @returns the capacity of the underlying buffer in samples. */
  get cap(): number {
    return this.dataType.density.length(this.byteCap);
  }

  /** @returns the length of the underlying buffer in samples. */
  get byteLength(): Size {
    if (this.pos === FULL_BUFFER) return this.byteCap;
    return this.dataType.density.size(this.pos);
  }

  /** @returns the number of samples in this array. */
  get length(): number {
    if (this.pos === FULL_BUFFER) return this.data.length;
    return this.pos;
  }

  /**
   * Creates a new array with a different data type.
   * @param target the data type to convert to.
   * @param sampleOffset an offset to apply to each sample. This can help with precision
   * issues when converting between data types.
   *
   * WARNING: This method is expensive and copies the entire underlying array. There
   * also may be untimely precision issues when converting between data types.
   */
  convert(target: DataType, sampleOffset: SampleValue = 0): LazyArray {
    if (this.dataType.equals(target)) return this;
    const data = new target.Array(this.length);
    for (let i = 0; i < this.length; i++) {
      data[i] = convertDataType(this.dataType, target, this.data[i], sampleOffset);
    }
    return new LazyArray(data.buffer, target, this._timeRange, sampleOffset);
  }

  /** @returns the maximum value in the array */
  get max(): SampleValue {
    if (this.pos === 0) return addSamples(0, this.sampleOffset);
    else if (this._max == null) {
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
    return addSamples(this._max, this.sampleOffset);
  }

  /** @returns the minimum value in the array */
  get min(): SampleValue {
    if (this.pos === 0) return addSamples(0, this.sampleOffset);
    else if (this._min == null) {
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
    return addSamples(this._min, this.sampleOffset);
  }

  /** @returns the bounds of this array. */
  get bounds(): Bounds {
    return new Bounds(Number(this.min), Number(this.max));
  }

  private maybeRecomputeMinMax(update: LazyArray): void {
    if (this._min != null && update.min < this._min) this._min = update.min;
    if (this._max != null && update.max > this._max) this._max = update.max;
  }

  enrich(): void {
    let _ = this.max;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    _ = this.min;
  }

  get range(): SampleValue {
    return addSamples(this.max, -this.min);
  }

  /**
   * @returns the index of the first sample that is greater than or equal to the given value.
   * The underlying array must be sorted. If it is not, the behavior of this method is undefined.
   * @param value the value to search for.
   */
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

  updateGLBuffer(gl: GLBufferController): void {
    if (!this.dataType.equals(DataType.FLOAT32))
      throw new Error("Only FLOAT32 arrays can be used in WebGL");
    const { buffer, bufferUsage, prevBuffer } = this.gl;

    // If no buffer has been created yet, create one.
    if (buffer == null) this.gl.buffer = gl.createBuffer();
    // If the current write position is the same as the previous buffer, we're already
    // up date.
    if (this.pos === prevBuffer) return;

    // Bind the buffer.
    gl.bindBuffer(gl.ARRAY_BUFFER, this.gl.buffer);

    // This means we only need to buffer part of the array.
    if (this.pos !== FULL_BUFFER) {
      if (prevBuffer === 0) {
        gl.bufferData(gl.ARRAY_BUFFER, this.byteCap.valueOf(), gl.STATIC_DRAW);
      }
      const byteOffset = this.dataType.density.size(prevBuffer).valueOf();
      const slice = this.underlyingData.slice(this.gl.prevBuffer, this.pos);
      gl.bufferSubData(gl.ARRAY_BUFFER, byteOffset, slice.buffer);
      this.gl.prevBuffer = this.pos;
    } else {
      // This means we can buffer the entire array in a single go.
      gl.bufferData(
        gl.ARRAY_BUFFER,
        this.buffer,
        bufferUsage === "static" ? gl.STATIC_DRAW : gl.DYNAMIC_DRAW
      );
      this.gl.prevBuffer = FULL_BUFFER;
    }
  }

  private maybeGarbageCollectGLBuffer(gl: GLBufferController): void {
    if (this.gl.buffer == null) return;
    gl.deleteBuffer(this.gl.buffer);
  }

  get glBuffer(): WebGLBuffer {
    if (this.gl.buffer == null) throw new Error("gl buffer not initialized");
    if (!(this.gl.prevBuffer === this.pos)) console.warn("buffer not updated");
    return this.gl.buffer;
  }

  slice(start: number, end?: number): LazyArray {
    const d = this.data.slice(start, end);
    return new LazyArray(d, this.dataType, TimeRange.ZERO, this.sampleOffset);
  }
}

export const addSamples = (a: SampleValue, b: SampleValue): SampleValue => {
  if (typeof a === "bigint" && typeof b === "bigint") return a + b;
  if (typeof a === "number" && typeof b === "number") return a + b;
  return Number(a) + Number(b);
};
