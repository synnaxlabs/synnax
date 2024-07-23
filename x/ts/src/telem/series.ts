// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { binary } from "@/binary";
import { caseconv } from "@/caseconv";
import { compare } from "@/compare";
import { id } from "@/id";
import { bounds } from "@/spatial";
import { type GLBufferController, type GLBufferUsage } from "@/telem/gl";
import {
  convertDataType,
  type CrudeDataType,
  type CrudeTimeStamp,
  DataType,
  isTelemValue,
  type NumericTelemValue,
  type Rate,
  Size,
  type TelemValue,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type TypedArray,
} from "@/telem/telem";

interface GL {
  control: GLBufferController | null;
  buffer: WebGLBuffer | null;
  prevBuffer: number;
  bufferUsage: GLBufferUsage;
}

export interface SeriesDigest {
  key: string;
  dataType: string;
  sampleOffset: NumericTelemValue;
  alignment: bounds.Bounds<bigint>;
  timeRange?: string;
  length: number;
  capacity: number;
}

interface BaseSeriesProps {
  dataType?: CrudeDataType;
  timeRange?: TimeRange;
  sampleOffset?: NumericTelemValue;
  glBufferUsage?: GLBufferUsage;
  alignment?: bigint;
  key?: string;
}

export type CrudeSeries =
  | Series
  | ArrayBuffer
  | TypedArray
  | string[]
  | number[]
  | boolean[]
  | unknown[]
  | TimeStamp[]
  | Date[]
  | TelemValue;

export const isCrudeSeries = (value: unknown): value is CrudeSeries => {
  if (value == null) return false;
  if (Array.isArray(value)) return true;
  if (value instanceof ArrayBuffer) return true;
  if (ArrayBuffer.isView(value) && !(value instanceof DataView)) return true;
  if (value instanceof Series) return true;
  return isTelemValue(value);
};

export interface SeriesProps extends BaseSeriesProps {
  data: CrudeSeries;
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
export class Series<T extends TelemValue = TelemValue> {
  key: string = "";
  isSynnaxSeries = true;
  /** The data type of the array */
  readonly dataType: DataType;
  /**
   * A sample offset that can be used to shift the values of all samples upwards or
   * downwards. Typically used to convert arrays to lower precision while preserving
   * the relative range of actual values.
   */
  sampleOffset: NumericTelemValue;
  /**
   * Stores information about the buffer state of this array into a WebGL buffer.
   */
  private readonly gl: GL;
  /** The underlying data. */
  private readonly _data: ArrayBufferLike;
  readonly _timeRange?: TimeRange;
  readonly alignment: bigint = 0n;
  /** A cached minimum value. */
  private _cachedMin?: NumericTelemValue;
  /** A cached maximum value. */
  private _cachedMax?: NumericTelemValue;
  /** The write position of the buffer. */
  private writePos: number = FULL_BUFFER;
  /** Tracks the number of entities currently using this array. */
  private _refCount: number = 0;
  private _cachedLength?: number;

  constructor(props: SeriesProps | CrudeSeries) {
    if (isCrudeSeries(props)) props = { data: props };
    const {
      dataType,
      timeRange,
      sampleOffset = 0,
      glBufferUsage = "static",
      alignment = 0n,
      key = id.id(),
    } = props;
    const { data } = props;

    if (
      data instanceof Series ||
      (typeof data === "object" &&
        "isSynnaxSeries" in data &&
        data.isSynnaxSeries === true)
    ) {
      const data_ = data as Series;
      this.key = data_.key;
      this.dataType = data_.dataType;
      this.sampleOffset = data_.sampleOffset;
      this.gl = data_.gl;
      this._data = data_._data;
      this._timeRange = data_._timeRange;
      this.alignment = data_.alignment;
      this._cachedMin = data_._cachedMin;
      this._cachedMax = data_._cachedMax;
      this.writePos = data_.writePos;
      this._refCount = data_._refCount;
      this._cachedLength = data_._cachedLength;
      return;
    }
    const isSingle = isTelemValue(data);
    const isArray = Array.isArray(data);

    if (dataType != null) this.dataType = new DataType(dataType);
    else {
      if (data instanceof ArrayBuffer)
        throw new Error(
          "cannot infer data type from an ArrayBuffer instance when constructing a Series. Please provide a data type.",
        );
      else if (isArray || isSingle) {
        let first: TelemValue | unknown = data as TelemValue;
        if (!isSingle) {
          if (data.length === 0)
            throw new Error(
              "cannot infer data type from a zero length JS array when constructing a Series. Please provide a data type.",
            );
          first = data[0];
        }
        if (typeof first === "string") this.dataType = DataType.STRING;
        else if (typeof first === "number") this.dataType = DataType.FLOAT64;
        else if (typeof first === "bigint") this.dataType = DataType.INT64;
        else if (typeof first === "boolean") this.dataType = DataType.BOOLEAN;
        else if (
          first instanceof TimeStamp ||
          first instanceof Date ||
          first instanceof TimeStamp
        )
          this.dataType = DataType.TIMESTAMP;
        else if (typeof first === "object") this.dataType = DataType.JSON;
        else
          throw new Error(
            `cannot infer data type of ${typeof first} when constructing a Series from a JS array`,
          );
      } else this.dataType = new DataType(data);
    }

    if (!isArray && !isSingle) this._data = data;
    else {
      let data_ = isSingle ? [data] : data;
      const first = data_[0];
      if (
        first instanceof TimeStamp ||
        first instanceof Date ||
        first instanceof TimeSpan
      )
        data_ = data_.map((v) => new TimeStamp(v as CrudeTimeStamp).valueOf());
      if (this.dataType.equals(DataType.STRING)) {
        this._cachedLength = data_.length;
        this._data = new TextEncoder().encode(data_.join("\n") + "\n");
      } else if (this.dataType.equals(DataType.JSON)) {
        this._cachedLength = data_.length;
        this._data = new TextEncoder().encode(
          data_.map((d) => binary.JSON_ECD.encodeString(d)).join("\n") + "\n",
        );
      } else this._data = new this.dataType.Array(data_ as number[] & bigint[]).buffer;
    }

    this.key = key;
    this.alignment = alignment;
    this.sampleOffset = sampleOffset ?? 0;
    this._timeRange = timeRange;
    this.gl = {
      control: null,
      buffer: null,
      prevBuffer: 0,
      bufferUsage: glBufferUsage,
    };
  }

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
      data.map((d) => binary.JSON_ECD.encodeString(d)).join("\n") + "\n",
    );
    return new Series({ data: buffer, dataType: DataType.JSON, timeRange });
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
    this.underlyingData.set(
      toWrite.data as unknown as ArrayLike<bigint> & ArrayLike<number>,
      this.writePos,
    );
    this.maybeRecomputeMinMax(toWrite);
    this._cachedLength = undefined;
    this.writePos += toWrite.length;
    return toWrite.length;
  }

  /** @returns the underlying buffer backing this array. */
  get buffer(): ArrayBufferLike {
    return this._data;
  }

  private get underlyingData(): TypedArray {
    return new this.dataType.Array(this._data);
  }

  /** @returns a native typed array with the proper data type. */
  get data(): TypedArray {
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
      .map((s) => schema.parse(binary.JSON_ECD.decodeString(s)));
  }

  /** @returns the time range of this array. */
  get timeRange(): TimeRange {
    if (this._timeRange == null) throw new Error("time range not set on series");
    return this._timeRange;
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
    if (this._cachedLength != null) return this._cachedLength;
    if (this.dataType.isVariable) return this.calculateCachedLength();
    if (this.writePos === FULL_BUFFER) return this.data.length;
    return this.writePos;
  }

  private calculateCachedLength(): number {
    if (!this.dataType.isVariable)
      throw new Error("cannot calculate length of a non-variable length data type");
    let cl = 0;
    this.data.forEach((v) => {
      if (v === 10) cl++;
    });
    this._cachedLength = cl;
    return cl;
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
  convert(target: DataType, sampleOffset: NumericTelemValue = 0): Series {
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

  private calcRawMax(): NumericTelemValue {
    if (this.length === 0) return -Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP)) {
      this._cachedMax = this.data[this.data.length - 1];
    } else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this._cachedMax = d.reduce((a, b) => (a > b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this._cachedMax = d.reduce((a, b) => (a > b ? a : b));
    }
    return this._cachedMax;
  }

  /** @returns the maximum value in the array */
  get max(): NumericTelemValue {
    if (this.dataType.isVariable)
      throw new Error("cannot calculate maximum on a variable length data type");
    if (this.writePos === 0) return -Infinity;
    else if (this._cachedMax == null) this._cachedMax = this.calcRawMax();
    return addSamples(this._cachedMax, this.sampleOffset);
  }

  private calcRawMin(): NumericTelemValue {
    if (this.length === 0) return Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP)) {
      this._cachedMin = this.data[0];
    } else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this._cachedMin = d.reduce((a, b) => (a < b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this._cachedMin = d.reduce((a, b) => (a < b ? a : b));
    }
    return this._cachedMin;
  }

  /** @returns the minimum value in the array */
  get min(): NumericTelemValue {
    if (this.dataType.isVariable)
      throw new Error("cannot calculate minimum on a variable length data type");
    if (this.writePos === 0) return Infinity;
    else if (this._cachedMin == null) this._cachedMin = this.calcRawMin();
    return addSamples(this._cachedMin, this.sampleOffset);
  }

  /** @returns the bounds of this array. */
  get bounds(): bounds.Bounds {
    return bounds.construct(Number(this.min), Number(this.max));
  }

  private maybeRecomputeMinMax(update: Series): void {
    if (this._cachedMin != null) {
      const min = update._cachedMin ?? update.calcRawMin();
      if (min < this._cachedMin) this._cachedMin = min;
    }
    if (this._cachedMax != null) {
      const max = update._cachedMax ?? update.calcRawMax();
      if (max > this._cachedMax) this._cachedMax = max;
    }
  }

  enrich(): void {
    let _ = this.max;

    _ = this.min;
  }

  get range(): NumericTelemValue {
    return addSamples(this.max, -this.min);
  }

  at(index: number, required: true): T;

  at(index: number, required?: false): T | undefined;

  at(index: number, required?: boolean): T | undefined {
    if (this.dataType.isVariable) return this.atVariable(index, required ?? false);
    if (index < 0) index = this.length + index;
    const v = this.data[index];
    if (v == null) {
      if (required === true) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
    return addSamples(v, this.sampleOffset) as T;
  }

  private atVariable(index: number, required: boolean): T | undefined {
    if (index < 0) index = this.length + index;
    let start = 0;
    let end = 0;
    for (let i = 0; i < this.data.length; i++) {
      if (this.data[i] === 10) {
        if (index === 0) {
          end = i;
          break;
        }
        start = i + 1;
        index--;
      }
    }
    if (end === 0) end = this.data.length;
    if (start >= end || index > 0) {
      if (required) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
    const slice = this.data.slice(start, end);
    if (this.dataType.equals(DataType.STRING))
      return new TextDecoder().decode(slice) as unknown as T;
    return caseconv.snakeToCamel(
      JSON.parse(new TextDecoder().decode(slice)),
    ) as unknown as T;
  }

  /**
   * @returns the index of the first sample that is greater than or equal to the given value.
   * The underlying array must be sorted. If it is not, the behavior of this method is undefined.
   * @param value the value to search for.
   */
  binarySearch(value: NumericTelemValue): number {
    let left = 0;
    let right = this.length - 1;
    const cf = compare.newF(value);
    while (left <= right) {
      const mid = Math.floor((left + right) / 2);
      const cmp = cf(this.at(mid, true) as NumericTelemValue, value);
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

  as(jsType: "string"): Series<string>;

  as(jsType: "number"): Series<number>;

  as(jsType: "bigint"): Series<bigint>;

  as<T extends TelemValue>(jsType: "string" | "number" | "bigint"): Series<T> {
    if (jsType === "string") {
      if (!this.dataType.equals(DataType.STRING))
        throw new Error(
          `cannot convert series of type ${this.dataType.toString()} to string`,
        );
      return this as unknown as Series<T>;
    }
    if (jsType === "number") {
      if (!this.dataType.isNumeric)
        throw new Error(
          `cannot convert series of type ${this.dataType.toString()} to number`,
        );
      return this as unknown as Series<T>;
    }
    if (jsType === "bigint") {
      if (!this.dataType.equals(DataType.INT64))
        throw new Error(
          `cannot convert series of type ${this.dataType.toString()} to bigint`,
        );
      return this as unknown as Series<T>;
    }
    throw new Error(`cannot convert series to ${jsType as string}`);
  }

  get digest(): SeriesDigest {
    return {
      key: this.key,
      dataType: this.dataType.toString(),
      sampleOffset: this.sampleOffset,
      alignment: this.alignmentBounds,
      timeRange: this._timeRange?.toString(),
      length: this.length,
      capacity: this.capacity,
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

  get alignmentBounds(): bounds.Bounds<bigint> {
    return bounds.construct(this.alignment, this.alignment + BigInt(this.length));
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

  [Symbol.iterator](): Iterator<T> {
    if (this.dataType.isVariable) {
      const s = new StringSeriesIterator(this);
      if (this.dataType.equals(DataType.JSON)) {
        return new JSONSeriesIterator(s) as Iterator<T>;
      }
      return s as Iterator<T>;
    }
    return new FixedSeriesIterator(this) as Iterator<T>;
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
      alignment: this.alignment + BigInt(start),
    });
  }

  reAlign(alignment: bigint): Series {
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

class StringSeriesIterator implements Iterator<string> {
  private readonly series: Series;
  private index: number;
  private readonly decoder: TextDecoder;

  constructor(series: Series) {
    if (!series.dataType.isVariable)
      throw new Error(
        "cannot create a variable series iterator for a non-variable series",
      );
    this.series = series;
    this.index = 0;
    this.decoder = new TextDecoder();
  }

  next(): IteratorResult<string> {
    const start = this.index;
    const data = this.series.data;
    while (this.index < data.length && data[this.index] !== 10) this.index++;
    const end = this.index;
    if (start === end) return { done: true, value: undefined };
    this.index++;
    const s = this.decoder.decode(this.series.buffer.slice(start, end));
    return { done: false, value: s };
  }

  [Symbol.iterator](): Iterator<TelemValue> {
    return this;
  }
}

class JSONSeriesIterator implements Iterator<unknown> {
  private readonly wrapped: Iterator<string>;

  constructor(wrapped: Iterator<string>) {
    this.wrapped = wrapped;
  }

  next(): IteratorResult<object> {
    const next = this.wrapped.next();
    if (next.done === true) return { done: true, value: undefined };
    return {
      done: false,
      value: binary.JSON_ECD.decodeString(next.value),
    };
  }

  [Symbol.iterator](): Iterator<object> {
    return this;
  }

  [Symbol.toStringTag] = "JSONSeriesIterator";
}

class FixedSeriesIterator implements Iterator<NumericTelemValue> {
  series: Series;
  index: number;
  constructor(series: Series) {
    this.series = series;
    this.index = 0;
  }

  next(): IteratorResult<NumericTelemValue> {
    if (this.index >= this.series.length) return { done: true, value: undefined };
    return {
      done: false,
      value: this.series.at(this.index++, true) as NumericTelemValue,
    };
  }

  [Symbol.iterator](): Iterator<NumericTelemValue> {
    return this;
  }

  [Symbol.toStringTag] = "SeriesIterator";
}

export const addSamples = (
  a: NumericTelemValue,
  b: NumericTelemValue,
): NumericTelemValue => {
  if (typeof a === "bigint" && typeof b === "bigint") return a + b;
  if (typeof a === "number" && typeof b === "number") return a + b;
  if (b === 0) return a;
  if (a === 0) return b;
  return Number(a) + Number(b);
};

export class MultiSeries<T extends TelemValue = TelemValue> implements Iterable<T> {
  readonly series: Array<Series<T>>;

  constructor(series: Array<Series<T>>) {
    if (series.length !== 0) {
      const type = series[0].dataType;
      for (let i = 1; i < series.length; i++)
        if (!series[i].dataType.equals(type))
          throw new Error("[multi-series] - series must have the same data type");
    }
    this.series = series;
  }

  as(jsType: "string"): MultiSeries<string>;

  as(jsType: "number"): MultiSeries<number>;

  as(jsType: "bigint"): MultiSeries<bigint>;

  as<T extends TelemValue>(dataType: CrudeDataType): MultiSeries<T> {
    if (!new DataType(dataType).equals(this.dataType))
      throw new Error(
        `cannot convert series of type ${this.dataType.toString()} to ${dataType.toString()}`,
      );
    return this as unknown as MultiSeries<T>;
  }

  get dataType(): DataType {
    if (this.series.length === 0) return DataType.UNKNOWN;
    return this.series[0].dataType;
  }

  get timeRange(): TimeRange {
    if (this.series.length === 0) return TimeRange.ZERO;
    return new TimeRange(
      this.series[0].timeRange.start,
      this.series[this.series.length - 1].timeRange.end,
    );
  }

  push(series: Series<T>): void {
    this.series.push(series);
  }

  get length(): number {
    return this.series.reduce((a, b) => a + b.length, 0);
  }

  at(index: number, required: true): T;

  at(index: number, required?: false): T | undefined;

  at(index: number, required: boolean = false): T | undefined {
    if (index < 0) index = this.length + index;
    for (const ser of this.series) {
      if (index < ser.length) return ser.at(index, required as true);
      index -= ser.length;
    }
    if (required) throw new Error(`[series] - no value at index ${index}`);
    return undefined;
  }

  get byteLength(): Size {
    return new Size(this.series.reduce((a, b) => a + b.byteLength.valueOf(), 0));
  }

  get data(): TypedArray {
    const buf = new this.dataType.Array(this.length);
    let offset = 0;
    for (const ser of this.series) {
      buf.set(ser.data as ArrayLike<any>, offset);
      offset += ser.length;
    }
    return new this.dataType.Array(buf);
  }

  [Symbol.iterator](): Iterator<T> {
    if (this.series.length === 0)
      return {
        next(): IteratorResult<T> {
          return { done: true, value: undefined };
        },
      };
    return new MultiSeriesIterator<T>(this.series);
  }
}

class MultiSeriesIterator<T extends TelemValue = TelemValue> implements Iterator<T> {
  private readonly series: Array<Series<T>>;
  private seriesIndex: number;
  private internal: Iterator<T>;

  constructor(series: Array<Series<T>>) {
    this.series = series;
    this.seriesIndex = 0;
    this.internal = series[0][Symbol.iterator]();
  }

  next(): IteratorResult<T> {
    const next = this.internal.next();
    if (next.done === false) return next;
    if (this.seriesIndex === this.series.length - 1)
      return { done: true, value: undefined };
    this.internal = this.series[++this.seriesIndex][Symbol.iterator]();
    return this.next();
  }

  [Symbol.iterator](): Iterator<TelemValue | unknown> {
    return this;
  }

  [Symbol.toStringTag] = "MultiSeriesIterator";
}
