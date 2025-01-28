// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { binary } from "@/binary";
import { caseconv } from "@/caseconv";
import { compare } from "@/compare";
import { id } from "@/id";
import { type math } from "@/math";
import { bounds } from "@/spatial";
import {
  type GLBufferController,
  type GLBufferUsage,
  glBufferUsageZ,
} from "@/telem/gl";
import {
  convertDataType,
  type CrudeDataType,
  type CrudeTimeStamp,
  DataType,
  isTelemValue,
  type Rate,
  Size,
  type TelemValue,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type TypedArray,
} from "@/telem/telem";
import { zodutil } from "@/zodutil";

interface GL {
  control: GLBufferController | null;
  buffer: WebGLBuffer | null;
  prevBuffer: number;
  bufferUsage: GLBufferUsage;
}

export interface IterableIterator<T> extends Iterator<T>, Iterable<T> {}

export interface SeriesDigest {
  key: string;
  dataType: string;
  sampleOffset: math.Numeric;
  alignment: {
    lower: AlignmentDigest;
    upper: AlignmentDigest;
  };
  timeRange?: string;
  length: number;
  capacity: number;
}

interface BaseSeriesProps {
  dataType?: CrudeDataType;
  timeRange?: TimeRange;
  sampleOffset?: math.Numeric;
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
  data?: CrudeSeries | null;
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

const noopIterableIterator: IterableIterator<never> = {
  [Symbol.iterator]: () => noopIterableIterator,
  next: () => ({ done: true, value: undefined }),
};

const stringArrayZ = z.string().transform(
  (s) =>
    new Uint8Array(
      atob(s)
        .split("")
        .map((c) => c.charCodeAt(0)),
    ).buffer as ArrayBuffer,
);

const nullArrayZ = z
  .union([z.null(), z.undefined()])
  .transform(() => new Uint8Array().buffer as ArrayBuffer);

const NEW_LINE = 10;

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
  sampleOffset: math.Numeric;
  /**
   * Stores information about the buffer state of this array into a WebGL buffer.
   */
  private readonly gl: GL;
  /** The underlying data. */
  private readonly _data: ArrayBuffer;
  readonly _timeRange?: TimeRange;
  readonly alignment: bigint = 0n;
  /** A cached minimum value. */
  private _cachedMin?: math.Numeric;
  /** A cached maximum value. */
  private _cachedMax?: math.Numeric;
  /** The write position of the buffer. */
  private writePos: number = FULL_BUFFER;
  /** Tracks the number of entities currently using this array. */
  private _refCount: number = 0;
  /** Caches the length of the array for variable length data types. */
  private _cachedLength?: number;
  /** Caches the indexes of the array for variable length data types. */
  private _cachedIndexes?: number[];

  static readonly crudeZ = z.object({
    timeRange: TimeRange.z.optional(),
    dataType: DataType.z,
    alignment: zodutil.bigInt.optional(),
    data: z.union([
      stringArrayZ,
      nullArrayZ,
      z.instanceof(ArrayBuffer),
      z.instanceof(Uint8Array),
    ]),
    glBufferUsage: glBufferUsageZ.optional().default("static").optional(),
  });

  static readonly z = Series.crudeZ.transform((props) => new Series(props));

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
    const data = props.data ?? [];
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
    else if (data instanceof ArrayBuffer)
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

    if (!isArray && !isSingle) this._data = data as ArrayBuffer;
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
        this._data = new TextEncoder().encode(`${data_.join("\n")}\n`)
          .buffer as ArrayBuffer;
      } else if (this.dataType.equals(DataType.JSON)) {
        this._cachedLength = data_.length;
        this._data = new TextEncoder().encode(
          `${data_.map((d) => binary.JSON_CODEC.encodeString(d)).join("\n")}\n`,
        ).buffer as ArrayBuffer;
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
    for (let i = 0; i < length; i++)
      data[i] = BigInt(start.add(rate.span(i)).valueOf());
    return new Series({ data, dataType: DataType.TIMESTAMP, timeRange });
  }

  get refCount(): number {
    return this._refCount;
  }

  static fromStrings(data: string[], timeRange?: TimeRange): Series {
    const buffer = new TextEncoder().encode(
      `${data.join("\n")}\n`,
    ) as Uint8Array<ArrayBuffer>;
    return new Series({ data: buffer, dataType: DataType.STRING, timeRange });
  }

  static fromJSON<T>(data: T[], timeRange?: TimeRange): Series {
    const buffer = new TextEncoder().encode(
      `${data.map((d) => binary.JSON_CODEC.encodeString(d)).join("\n")}\n`,
    ) as Uint8Array<ArrayBuffer>;
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
    if (this.dataType.isVariable) return this.writeVariable(other);
    return this.writeFixed(other);
  }

  private writeVariable(other: Series): number {
    if (this.writePos === FULL_BUFFER) return 0;
    const available = this.byteCapacity.valueOf() - this.writePos;
    const toWrite = other.subBytes(0, available);
    this.writeToUnderlyingData(toWrite);
    this.writePos += toWrite.byteLength.valueOf();
    if (this._cachedLength != null) {
      this._cachedLength += toWrite.length;
      this.calculateCachedLength();
    }
    return toWrite.length;
  }

  private writeFixed(other: Series): number {
    if (this.writePos === FULL_BUFFER) return 0;
    const available = this.capacity - this.writePos;
    const toWrite = other.sub(0, available);
    this.writeToUnderlyingData(toWrite);
    this._cachedLength = undefined;
    this.maybeRecomputeMinMax(toWrite);
    this.writePos += toWrite.length;
    return toWrite.length;
  }

  private writeToUnderlyingData(data: Series) {
    this.underlyingData.set(
      data.data as unknown as ArrayLike<bigint> & ArrayLike<number>,
      this.writePos,
    );
  }

  /** @returns the underlying buffer backing this array. */
  get buffer(): ArrayBuffer {
    if (typeof this._data === "object" && "buffer" in this._data)
      return (this._data as unknown as Uint8Array).buffer as ArrayBuffer;
    return this._data;
  }

  private get underlyingData(): TypedArray {
    return new this.dataType.Array(this._data);
  }

  /** @returns a native typed array with the proper data type. */
  get data(): TypedArray {
    if (this.writePos === FULL_BUFFER) return this.underlyingData;
    // @ts-expect-error - ABC
    return new this.dataType.Array(this._data, 0, this.writePos);
  }

  toStrings(): string[] {
    if (!this.dataType.matches(DataType.STRING, DataType.UUID))
      throw new Error("cannot convert non-string series to strings");
    return new TextDecoder().decode(this.underlyingData).split("\n").slice(0, -1);
  }

  toUUIDs(): string[] {
    if (!this.dataType.equals(DataType.UUID))
      throw new Error("cannot convert non-uuid series to uuids");
    const den = DataType.UUID.density.valueOf();
    const r = Array(this.length);

    for (let i = 0; i < this.length; i++) {
      const v = this.underlyingData.slice(i * den, (i + 1) * den);
      const id = Array.from(new Uint8Array(v.buffer), (b) =>
        b.toString(16).padStart(2, "0"),
      )
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
      .decode(this.underlyingData)
      .split("\n")
      .slice(0, -1)
      .map((s) => schema.parse(binary.JSON_CODEC.decodeString(s)));
  }

  /** @returns the time range of this array. */
  get timeRange(): TimeRange {
    if (this._timeRange == null) throw new Error("time range not set on series");
    return this._timeRange;
  }

  /** @returns the capacity of the series in bytes. */
  get byteCapacity(): Size {
    return new Size(this.underlyingData.byteLength);
  }

  /** @returns the capacity of the series in samples. */
  get capacity(): number {
    if (this.dataType.isVariable) return this.byteCapacity.valueOf();
    return this.dataType.density.length(this.byteCapacity);
  }

  /** @returns the length of the series in bytes. */
  get byteLength(): Size {
    if (this.writePos === FULL_BUFFER) return this.byteCapacity;
    if (this.dataType.isVariable) return new Size(this.writePos);
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
    const ci: number[] = [0];
    this.data.forEach((v, i) => {
      if (v !== NEW_LINE) return;
      cl++;
      ci.push(i + 1);
    });
    this._cachedIndexes = ci;
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
  convert(target: DataType, sampleOffset: math.Numeric = 0): Series {
    if (this.dataType.equals(target)) return this;
    const data = new target.Array(this.length);
    for (let i = 0; i < this.length; i++)
      data[i] = convertDataType(this.dataType, target, this.data[i], sampleOffset);
    return new Series({
      data: data.buffer,
      dataType: target,
      timeRange: this._timeRange,
      sampleOffset,
      glBufferUsage: this.gl.bufferUsage,
      alignment: this.alignment,
    });
  }

  private calcRawMax(): math.Numeric {
    if (this.length === 0) return -Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP))
      this._cachedMax = this.data[this.data.length - 1];
    else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this._cachedMax = d.reduce((a, b) => (a > b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this._cachedMax = d.reduce((a, b) => (a > b ? a : b));
    }
    return this._cachedMax;
  }

  /** @returns the maximum value in the array */
  get max(): math.Numeric {
    if (this.dataType.isVariable)
      throw new Error("cannot calculate maximum on a variable length data type");
    if (this.writePos === 0) return -Infinity;
    this._cachedMax ??= this.calcRawMax();
    return addSamples(this._cachedMax, this.sampleOffset);
  }

  private calcRawMin(): math.Numeric {
    if (this.length === 0) return Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP)) this._cachedMin = this.data[0];
    else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this._cachedMin = d.reduce((a, b) => (a < b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this._cachedMin = d.reduce((a, b) => (a < b ? a : b));
    }
    return this._cachedMin;
  }

  /** @returns the minimum value in the array */
  get min(): math.Numeric {
    if (this.dataType.isVariable)
      throw new Error("cannot calculate minimum on a variable length data type");
    if (this.writePos === 0) return Infinity;
    this._cachedMin ??= this.calcRawMin();
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

  get range(): math.Numeric {
    return addSamples(this.max, -this.min);
  }

  atAlignment(alignment: bigint, required: true): T;

  atAlignment(alignment: bigint, required?: false): T | undefined;

  atAlignment(alignment: bigint, required?: boolean): T | undefined {
    const index = Number(alignment - this.alignment);
    if (index < 0 || index >= this.length) {
      if (required === true) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
    return this.at(index, required as true);
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
    let start = 0;
    let end = 0;
    if (this._cachedIndexes != null) {
      start = this._cachedIndexes[index];
      end = this._cachedIndexes[index + 1] - 1;
    } else {
      if (index < 0) index = this.length + index;
      for (let i = 0; i < this.data.length; i++)
        if (this.data[i] === NEW_LINE) {
          if (index === 0) {
            end = i;
            break;
          }
          start = i + 1;
          index--;
        }
      if (end === 0) end = this.data.length;
      if (start >= end || index > 0) {
        if (required) throw new Error(`[series] - no value at index ${index}`);
        return undefined;
      }
    }
    const slice = this.data.slice(start, end);
    if (this.dataType.equals(DataType.STRING))
      return new TextDecoder().decode(slice) as T;
    return caseconv.snakeToCamel(JSON.parse(new TextDecoder().decode(slice))) as T;
  }

  /**
   * @returns the index of the first sample that is greater than or equal to the given value.
   * The underlying array must be sorted. If it is not, the behavior of this method is undefined.
   * @param value the value to search for.
   */
  binarySearch(value: math.Numeric): number {
    let left = 0;
    let right = this.length - 1;
    const cf = compare.newF(value);
    while (left <= right) {
      const mid = Math.floor((left + right) / 2);
      const cmp = cf(this.at(mid, true) as math.Numeric, value);
      if (cmp === 0) return mid;
      if (cmp < 0) left = mid + 1;
      else right = mid - 1;
    }
    return left;
  }

  updateGLBuffer(gl: GLBufferController): void {
    this.gl.control = gl;
    if (
      !this.dataType.equals(DataType.FLOAT32) &&
      !this.dataType.equals(DataType.UINT8)
    )
      throw new Error("Only FLOAT32 and UINT8 arrays can be used in WebGL");
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
      if (prevBuffer === 0)
        gl.bufferData(gl.ARRAY_BUFFER, this.byteCapacity.valueOf(), gl.STATIC_DRAW);
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
      alignment: {
        lower: alignmentDigest(this.alignmentBounds.lower),
        upper: alignmentDigest(this.alignmentBounds.upper),
      },
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
      if (this.dataType.equals(DataType.JSON))
        return new JSONSeriesIterator(s) as Iterator<T>;
      return s as Iterator<T>;
    }
    return new FixedSeriesIterator(this) as Iterator<T>;
  }

  slice(start: number, end?: number): Series {
    return this.sliceSub(false, start, end);
  }

  sub(start: number, end?: number): Series {
    return this.sliceSub(true, start, end);
  }

  subIterator(start: number, end?: number): IterableIterator<T> {
    return new SubIterator(this, start, end ?? this.length);
  }

  subAlignmentIterator(start: bigint, end: bigint): IterableIterator<T> {
    return new SubIterator(
      this,
      Number(start - this.alignment),
      Number(end - this.alignment),
    );
  }

  private subBytes(start: number, end?: number): Series {
    if (start >= 0 && (end == null || end >= this.byteLength.valueOf())) return this;
    const data = this.data.subarray(start, end);
    return new Series({
      data,
      dataType: this.dataType,
      timeRange: this._timeRange,
      sampleOffset: this.sampleOffset,
      glBufferUsage: this.gl.bufferUsage,
      alignment: this.alignment + BigInt(start),
    });
  }

  private sliceSub(sub: boolean, start: number, end?: number): Series {
    if (start <= 0 && (end == null || end >= this.length)) return this;
    let data: TypedArray;
    if (sub) data = this.data.subarray(start, end);
    else data = this.data.slice(start, end);
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

class SubIterator<T> implements Iterator<T>, Iterable<T> {
  private readonly series: Series;
  private readonly end: number;
  private index: number;

  constructor(series: Series, start: number, end: number) {
    this.series = series;
    const b = bounds.construct(0, series.length);
    this.end = bounds.clamp(b, end);
    this.index = bounds.clamp(b, start);
  }

  next(): IteratorResult<T> {
    if (this.index >= this.end) return { done: true, value: undefined };
    return { done: false, value: this.series.at(this.index++, true) as T };
  }

  [Symbol.iterator](): Iterator<T> {
    return this;
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
    while (this.index < data.length && data[this.index] !== NEW_LINE) this.index++;
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
      value: binary.JSON_CODEC.decodeString(next.value),
    };
  }

  [Symbol.iterator](): Iterator<object> {
    return this;
  }

  [Symbol.toStringTag] = "JSONSeriesIterator";
}

class FixedSeriesIterator implements Iterator<math.Numeric> {
  series: Series;
  index: number;
  constructor(series: Series) {
    this.series = series;
    this.index = 0;
  }

  next(): IteratorResult<math.Numeric> {
    if (this.index >= this.series.length) return { done: true, value: undefined };
    return {
      done: false,
      value: this.series.at(this.index++, true) as math.Numeric,
    };
  }

  [Symbol.iterator](): Iterator<math.Numeric> {
    return this;
  }

  [Symbol.toStringTag] = "SeriesIterator";
}

export const addSamples = (a: math.Numeric, b: math.Numeric): math.Numeric => {
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

  get alignment(): bigint {
    if (this.series.length === 0) return 0n;
    return this.series[0].alignment;
  }

  get alignmentBounds(): bounds.Bounds<bigint> {
    if (this.series.length === 0) return bounds.construct(0n, 0n);
    return bounds.construct(
      this.series[0].alignmentBounds.lower,
      this.series[this.series.length - 1].alignmentBounds.upper,
    );
  }

  push(series: Series<T>): void {
    this.series.push(series);
  }

  get length(): number {
    return this.series.reduce((a, b) => a + b.length, 0);
  }

  atAlignment(alignment: bigint, required: true): T;

  atAlignment(alignment: bigint, required?: false): T | undefined;

  atAlignment(alignment: bigint, required?: boolean): T | undefined {
    if (this.series.length === 0) {
      if (required) throw new Error(`[series] - no value at alignment ${alignment}`);
      return undefined;
    }
    for (const ser of this.series)
      if (bounds.contains(ser.alignmentBounds, alignment))
        return ser.atAlignment(alignment, required as true);
    if (required) throw new Error(`[series] - no value at alignment ${alignment}`);
    return undefined;
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

  subIterator(start: number, end?: number): IterableIterator<T> {
    return new MultiSubIterator(this, start, end ?? this.length);
  }

  subAlignmentIterator(start: bigint, end: bigint): IterableIterator<T> {
    if (start >= this.alignmentBounds.upper || end <= this.alignmentBounds.lower)
      return noopIterableIterator;
    let startIdx = 0;
    for (let i = 0; i < this.series.length; i++) {
      const ser = this.series[i];
      if (start < ser.alignment) break;
      else if (start >= ser.alignmentBounds.upper) startIdx += ser.length;
      else if (bounds.contains(ser.alignmentBounds, start)) {
        startIdx += Number(start - ser.alignment);
        break;
      }
    }
    let endIdx = 0;
    for (let i = 0; i < this.series.length; i++) {
      const ser = this.series[i];
      if (end < ser.alignment) break;
      else if (end >= ser.alignmentBounds.upper) endIdx += ser.length;
      else if (bounds.contains(ser.alignmentBounds, end)) {
        endIdx += Number(end - ser.alignment);
        break;
      }
    }
    return new MultiSubIterator(this, startIdx, endIdx);
  }

  subAlignmentSpanIterator(start: bigint, span: number): IterableIterator<T> {
    if (start >= this.alignmentBounds.upper) return noopIterableIterator;
    let startIdx = 0;
    for (let i = 0; i < this.series.length; i++) {
      const ser = this.series[i];
      if (start < ser.alignment) break;
      else if (start >= ser.alignmentBounds.upper) startIdx += ser.length;
      else if (bounds.contains(ser.alignmentBounds, start)) {
        startIdx += Number(start - ser.alignment);
        break;
      }
    }
    return new MultiSubIterator(this, startIdx, startIdx + span);
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
    return new this.dataType.Array(buf.buffer);
  }

  traverseAlignment(start: bigint, dist: bigint): bigint {
    const b = this.series.map((s) => s.alignmentBounds);
    return bounds.traverse(b, start, dist);
  }

  distance(start: bigint, end: bigint): bigint {
    const b = this.series.map((s) => s.alignmentBounds);
    return bounds.distance(b, start, end);
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

class MultiSubIterator<T extends TelemValue = TelemValue>
  implements IterableIterator<T>
{
  private readonly series: MultiSeries<T>;
  private index: number;
  private end: number;

  constructor(series: MultiSeries<T>, start: number, end: number) {
    this.series = series;
    this.end = end;
    this.index = start;
  }

  next(): IteratorResult<T> {
    if (this.index >= this.end) return { done: true, value: undefined };
    return { done: false, value: this.series.at(this.index++, true) as T };
  }

  [Symbol.iterator](): Iterator<T> {
    return this;
  }
}

interface AlignmentDigest {
  domain: bigint;
  sample: bigint;
}

export type SeriesPayload = z.infer<typeof Series.crudeZ>;

const alignmentDigest = (alignment: bigint): AlignmentDigest => {
  const domain = alignment >> 32n;
  const sample = alignment & 0xffffffffn;
  return { domain, sample };
};
