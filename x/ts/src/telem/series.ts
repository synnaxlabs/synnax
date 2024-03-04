// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { nanoid } from "nanoid/non-secure";
import { type z } from "zod";

import { compare } from "@/compare";
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

export interface SeriesDigest {
  key: string;
  dataType: string;
  sampleOffset: SampleValue;
  alignment: bounds.Bounds;
  timeRange?: string;
  length: number;
}

interface BaseSeriesProps {
  dataType?: CrudeDataType;
  timeRange?: TimeRange;
  sampleOffset?: SampleValue;
  glBufferUsage?: GLBufferUsage;
  alignment?: number;
  key?: string;
}

export interface SeriesProps extends BaseSeriesProps {
  data: ArrayBuffer | NativeTypedArray;
}

export interface SeriesAllocProps extends BaseSeriesProps {
  capacity: number;
  dataType: CrudeDataType;
}

const FULL_BUFFER = -1;

export interface SeriesMemInfo {
  key: string;
  length: number;
  byteLength: Size;
  glBuffer: boolean;
}

/**
 * Series is a strongly typed array of telemetry samples backed by an underlying binary
 * buffer.
 */
export class Series {
  key: string = "";
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

  static alloc({ capacity: length, dataType, ...props }: SeriesAllocProps): Series {
    if (length === 0)
      throw new Error("[Series] - cannot allocate an array of length 0");
    const data = new new DataType(dataType).Array(length);
    const arr = new Series({
      data: data.buffer,
      dataType,
      ...props,
    });
    arr.writePos = 0;
    return arr;
  }

  static generateTimestamps(length: number, rate: Rate, start: TimeStamp): Series {
    const timeRange = start.spanRange(rate.span(length));
    const data = new BigInt64Array(length);
    for (let i = 0; i < length; i++) {
      data[i] = BigInt(start.add(rate.span(i)).valueOf());
    }
    return new Series({ data, dataType: DataType.TIMESTAMP, timeRange });
  }

  get refCount(): number {
    return this._refCount;
  }

  static fromStrings(data: string[], timeRange?: TimeRange): Series {
    const buffer = new TextEncoder().encode(data.join("\n") + "\n");
    return new Series({ data: buffer, dataType: DataType.STRING, timeRange });
  }

  static fromJSON<T>(data: T[], timeRange?: TimeRange): Series {
    const buffer = new TextEncoder().encode(
      data.map((d) => JSON.stringify(d)).join("\n") + "\n",
    );
    return new Series({ data: buffer, dataType: DataType.JSON, timeRange });
  }

  constructor({
    data,
    dataType,
    timeRange,
    sampleOffset = 0,
    glBufferUsage = "static",
    alignment = 0,
    key = nanoid(),
  }: SeriesProps) {
    if (dataType == null && !(data instanceof ArrayBuffer)) {
      this.dataType = new DataType(data);
    } else if (dataType != null) {
      this.dataType = new DataType(dataType);
    } else {
      throw new Error(
        "must provide a data type when constructing a Series from a buffer",
      );
    }
    this.key = key;
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

  /**
   * Writes the given series to this series. If the series being written exceeds the 
   * remaining of series being written to, only the portion that fits will be written. 
   * @param other the series to write to this series. The data type of the series written
   * must be the same as the data type of the series being written to.
   * @returns the number of samples written. If the entire series fits, this value is
   * equal to the length of the series being written.
   */
  write(other: Series): number {
    if (!other.dataType.equals(this.dataType))
      throw new Error("buffer must be of the same type as this array");

    // We've filled the entire underlying buffer
    if (this.writePos === FULL_BUFFER) return 0;
    const available = this.capacity - this.writePos;

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

  toStrings(): string[] {
    if (!this.dataType.equals(DataType.STRING))
      throw new Error("cannot convert non-string series to strings");
    return new TextDecoder().decode(this.buffer).split("\n").slice(0, -1);
  }

  toUUIDs(): string[] {
    if (!this.dataType.equals(DataType.UUID))
      throw new Error("cannot convert non-uuid series to uuids");
    const den = DataType.UUID.density.valueOf();
    const r = Array(this.length);

    for (let i = 0; i < this.length; i++) {
      const v = this.buffer.slice(i * den, (i + 1) * den);
      const id = Array.from(new Uint8Array(v), (b) => b.toString(16).padStart(2, "0"))
        .join("")
        .replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, "$1-$2-$3-$4-$5");
      r[i] = id;
    }
    return r;
  }

  parseJSON<Z extends z.ZodTypeAny>(schema: Z): Array<z.output<Z>> {
    if (!this.dataType.equals(DataType.JSON))
      throw new Error("cannot convert non-string series to strings");
    return new TextDecoder()
      .decode(this.buffer)
      .split("\n")
      .slice(0, -1)
      .map((s) => schema.parse(JSON.parse(s)));
  }

  /** @returns the time range of this array. */
  get timeRange(): TimeRange {
    validateFieldNotNull("timeRange", this._timeRange);
    return this._timeRange!;
  }

  /** @returns the capacity of the series in bytes. */
  get byteCapacity(): Size {
    return new Size(this.buffer.byteLength);
  }

  /** @returns the capacity of the series in samples. */
  get capacity(): number {
    return this.dataType.density.length(this.byteCapacity);
  }

  /** @returns the length of the series in bytes. */
  get byteLength(): Size {
    if (this.writePos === FULL_BUFFER) return this.byteCapacity;
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
    return new Series({
      data: data.buffer,
      dataType: target,
      timeRange: this._timeRange,
      sampleOffset,
      glBufferUsage: this.gl.bufferUsage,
      alignment: this.alignment,
    });
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

  at(index: number, required: true): SampleValue;

  at(index: number, required?: false): SampleValue | undefined;

  at(index: number, required?: boolean): SampleValue | undefined {
    if (index < 0) index = this.length + index;
    const v = this.data[index];
    if (v == null) {
      if (required) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
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
    const cf = compare.newF(value);
    while (left <= right) {
      const mid = Math.floor((left + right) / 2);
      const cmp = cf(this.at(mid, true), value);
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
        gl.bufferData(gl.ARRAY_BUFFER, this.byteCapacity.valueOf(), gl.STATIC_DRAW);
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

  get digest(): SeriesDigest {
    return {
      key: this.key,
      dataType: this.dataType.toString(),
      sampleOffset: this.sampleOffset,
      alignment: this.alignmentBounds,
      timeRange: this._timeRange?.toString(),
      length: this.length,
    };
  }

  get memInfo(): SeriesMemInfo {
    return {
      key: this.key,
      length: this.length,
      byteLength: this.byteLength,
      glBuffer: this.gl.buffer != null,
    };
  }

  get alignmentBounds(): bounds.Bounds {
    return bounds.construct(this.alignment, this.alignment + this.length);
  }

  private maybeGarbageCollectGLBuffer(gl: GLBufferController): void {
    if (this.gl.buffer == null) return;
    gl.deleteBuffer(this.gl.buffer);
    this.gl.buffer = null;
    this.gl.prevBuffer = 0;
    this.gl.control = null;
  }

  get glBuffer(): WebGLBuffer {
    if (this.gl.buffer == null) throw new Error("gl buffer not initialized");
    if (!(this.gl.prevBuffer === this.writePos)) console.warn("buffer not updated");
    return this.gl.buffer;
  }

  slice(start: number, end?: number): Series {
    if (start <= 0 && (end == null || end >= this.length)) return this;
    const data = this.data.slice(start, end);
    return new Series({
      data,
      dataType: this.dataType,
      timeRange: this._timeRange,
      sampleOffset: this.sampleOffset,
      glBufferUsage: this.gl.bufferUsage,
      alignment: this.alignment + start,
    });
  }

  reAlign(alignment: number): Series {
    return new Series({
      data: this.buffer,
      dataType: this.dataType,
      timeRange: TimeRange.ZERO,
      sampleOffset: this.sampleOffset,
      glBufferUsage: "static",
      alignment,
    });
  }
}

export const addSamples = (a: SampleValue, b: SampleValue): SampleValue => {
  if (typeof a === "bigint" && typeof b === "bigint") return a + b;
  if (typeof a === "number" && typeof b === "number") return a + b;
  return Number(a) + Number(b);
};
