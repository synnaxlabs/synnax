// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Compare } from "@/compare";
import { bounds } from "@/spatial";
import { type GLBufferController, type GLBufferUsage } from "@/telem/gl";
import {
  convertDataType,
  DataType,
  type NativeTypedArray,
  type Rate,
  Size,
  TimeRange,
  type TimeStamp,
  type CrudeDataType,
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
 * Series is a strongly typed array of telemetry samples backed by an underlying binary
 * buffer.
 */
export class Series {
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
  private readonly _data: ArrayBufferLike;
  readonly _timeRange?: TimeRange;
  readonly alignment: number = 0;
  /** A cached minimum value. */
  private _min?: SampleValue;
  /** A cached maximum value. */
  private _max?: SampleValue;
  /** The write position of the buffer. */
  private writePos: number = FULL_BUFFER;
  /** Tracks the number of entities currently using this array. */
  private _refCount: number = 0;

  static alloc(
    length: number,
    dataType: CrudeDataType,
    timeRange?: TimeRange,
    sampleOffset?: SampleValue,
    glBufferUsage: GLBufferUsage = "static",
    alignment: number = 0,
  ): Series {
    if (length === 0)
      throw new Error("[Series] - cannot allocate an array of length 0");
    const data = new new DataType(dataType).Array(length);
    const arr = new Series(
      data.buffer,
      dataType,
      timeRange,
      sampleOffset,
      glBufferUsage,
      alignment,
    );
    arr.writePos = 0;
    return arr;
  }

  static generateTimestamps(length: number, rate: Rate, start: TimeStamp): Series {
    const tr = start.spanRange(rate.span(length));
    const data = new BigInt64Array(length);
    for (let i = 0; i < length; i++) {
      data[i] = BigInt(start.add(rate.span(i)).valueOf());
    }
    return new Series(data, DataType.TIMESTAMP, tr);
  }

  get refCount(): number {
    return this._refCount;
  }

  constructor(
    data: ArrayBuffer | NativeTypedArray,
    dataType?: CrudeDataType,
    timeRange?: TimeRange,
    sampleOffset?: SampleValue,
    glBufferUsage: GLBufferUsage = "static",
    alignment: number = 0,
  ) {
    if (dataType == null && !(data instanceof ArrayBuffer)) {
      this.dataType = new DataType(data);
    } else if (dataType != null) {
      this.dataType = new DataType(dataType);
    } else {
      throw new Error(
        "must provide a data type when constructing a Series from a buffer",
      );
    }
    this.alignment = alignment;
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
    this._refCount++;
    if (gl != null) this.updateGLBuffer(gl);
  }

  release(): void {
    this._refCount--;
    if (this._refCount === 0 && this.gl.control != null)
      this.maybeGarbageCollectGLBuffer(this.gl.control);
    else if (this._refCount < 0)
      throw new Error("cannot release an array with a negative reference count");
  }

  write(other: Series): number {
    if (!other.dataType.equals(this.dataType))
      throw new Error("buffer must be of the same type as this array");

    // We've filled the entire underlying buffer
    if (this.writePos === FULL_BUFFER) return 0;
    const available = this.cap - this.writePos;

    const toWrite = available < other.length ? other.slice(0, available) : other;
    this.underlyingData.set(toWrite.data as any, this.writePos);
    this.maybeRecomputeMinMax(toWrite);
    this.writePos += toWrite.length;
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
    if (this.writePos === FULL_BUFFER) return this.underlyingData;
    return new this.dataType.Array(this._data, 0, this.writePos);
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
    if (this.writePos === FULL_BUFFER) return this.byteCap;
    return this.dataType.density.size(this.writePos);
  }

  /** @returns the number of samples in this array. */
  get length(): number {
    if (this.writePos === FULL_BUFFER) return this.data.length;
    return this.writePos;
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
  convert(target: DataType, sampleOffset: SampleValue = 0): Series {
    if (this.dataType.equals(target)) return this;
    const data = new target.Array(this.length);
    for (let i = 0; i < this.length; i++) {
      data[i] = convertDataType(this.dataType, target, this.data[i], sampleOffset);
    }
    return new Series(
      data.buffer,
      target,
      this._timeRange,
      sampleOffset,
      this.gl.bufferUsage,
      this.alignment,
    );
  }

  private calcRawMax(): SampleValue {
    if (this.length === 0) return -Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP)) {
      this._max = this.data[this.data.length - 1];
    } else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this._max = d.reduce((a, b) => (a > b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this._max = d.reduce((a, b) => (a > b ? a : b));
    }
    return this._max;
  }

  /** @returns the maximum value in the array */
  get max(): SampleValue {
    if (this.writePos === 0) return -Infinity;
    else if (this._max == null) this._max = this.calcRawMax();
    return addSamples(this._max, this.sampleOffset);
  }

  private calcRawMin(): SampleValue {
    if (this.length === 0) return Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP)) {
      this._min = this.data[0];
    } else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this._min = d.reduce((a, b) => (a < b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this._min = d.reduce((a, b) => (a < b ? a : b));
    }
    return this._min;
  }

  /** @returns the minimum value in the array */
  get min(): SampleValue {
    if (this.writePos === 0) return Infinity;
    else if (this._min == null) this._min = this.calcRawMin();
    return addSamples(this._min, this.sampleOffset);
  }

  /** @returns the bounds of this array. */
  get bounds(): bounds.Bounds {
    return bounds.construct(Number(this.min), Number(this.max));
  }

  private maybeRecomputeMinMax(update: Series): void {
    if (this._min != null) {
      const min = update._min ?? update.calcRawMin();
      if (min < this._min) this._min = min;
    }
    if (this._max != null) {
      const max = update._max ?? update.calcRawMax();
      if (max > this._max) this._max = max;
    }
  }

  enrich(): void {
    let _ = this.max;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    _ = this.min;
  }

  get range(): SampleValue {
    return addSamples(this.max, -this.min);
  }

  at(index: number): SampleValue {
    const v = this.data[index];
    if (v == null) return undefined as any;
    return addSamples(v, this.sampleOffset);
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
      const cmp = compare(this.at(mid), value);
      if (cmp === 0) return mid;
      if (cmp < 0) left = mid + 1;
      else right = mid - 1;
    }
    return left;
  }

  updateGLBuffer(gl: GLBufferController): void {
    this.gl.control = gl;
    if (!this.dataType.equals(DataType.FLOAT32))
      throw new Error("Only FLOAT32 arrays can be used in WebGL");
    const { buffer, bufferUsage, prevBuffer } = this.gl;

    // If no buffer has been created yet, create one.
    if (buffer == null) this.gl.buffer = gl.createBuffer();
    // If the current write position is the same as the previous buffer, we're already
    // up date.
    if (this.writePos === prevBuffer) return;

    // Bind the buffer.
    gl.bindBuffer(gl.ARRAY_BUFFER, this.gl.buffer);

    // This means we only need to buffer part of the array.
    if (this.writePos !== FULL_BUFFER) {
      if (prevBuffer === 0) {
        gl.bufferData(gl.ARRAY_BUFFER, this.byteCap.valueOf(), gl.STATIC_DRAW);
      }
      const byteOffset = this.dataType.density.size(prevBuffer).valueOf();
      const slice = this.underlyingData.slice(this.gl.prevBuffer, this.writePos);
      gl.bufferSubData(gl.ARRAY_BUFFER, byteOffset, slice.buffer);
      this.gl.prevBuffer = this.writePos;
    } else {
      // This means we can buffer the entire array in a single go.
      gl.bufferData(
        gl.ARRAY_BUFFER,
        this.buffer,
        bufferUsage === "static" ? gl.STATIC_DRAW : gl.DYNAMIC_DRAW,
      );
      this.gl.prevBuffer = FULL_BUFFER;
    }
  }

  private maybeGarbageCollectGLBuffer(gl: GLBufferController): void {
    if (this.gl.buffer == null) return;
    gl.deleteBuffer(this.gl.buffer);
    this.gl.buffer = null;
    this.gl.prevBuffer = 0;
  }

  get glBuffer(): WebGLBuffer {
    if (this.gl.buffer == null) throw new Error("gl buffer not initialized");
    if (!(this.gl.prevBuffer === this.writePos)) console.warn("buffer not updated");
    return this.gl.buffer;
  }

  slice(start: number, end?: number): Series {
    const d = this.data.slice(start, end);
    return new Series(d, this.dataType, TimeRange.ZERO, this.sampleOffset);
  }
}

export const addSamples = (a: SampleValue, b: SampleValue): SampleValue => {
  if (typeof a === "bigint" && typeof b === "bigint") return a + b;
  if (typeof a === "number" && typeof b === "number") return a + b;
  return Number(a) + Number(b);
};
