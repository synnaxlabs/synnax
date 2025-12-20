// Copyright 2025 Synnax Labs, Inc.
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
import { instance } from "@/instance";
import { math } from "@/math";
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
  Size,
  type TelemValue,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type TypedArray,
} from "@/telem/telem";
import { uuid } from "@/uuid";

interface GL {
  control: GLBufferController | null;
  buffer: WebGLBuffer | null;
  prevBuffer: number;
  bufferUsage: GLBufferUsage;
}

interface IterableIterator<T> extends Iterator<T>, Iterable<T> {}

/** A condensed set of information describing the layout of a series. */
export interface SeriesDigest {
  key: string;
  dataType: string;
  sampleOffset: math.Numeric;
  alignment: {
    lower: AlignmentDigest;
    upper: AlignmentDigest;
    multiple: bigint;
  };
  timeRange?: string;
  length: number;
  capacity: number;
}

interface BaseSeriesArgs {
  dataType?: CrudeDataType;
  timeRange?: TimeRange;
  sampleOffset?: math.Numeric;
  glBufferUsage?: GLBufferUsage;
  alignment?: bigint;
  alignmentMultiple?: bigint;
  key?: string;
}

/** A value or set of values that a series can be constructed from. */
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

/** Arguments for constructing a {@link Series}. */
export interface SeriesArgs extends BaseSeriesArgs {
  data?: CrudeSeries | null;
}

/** Arguments for allocating a {@link Series} with a given capacity and data type. */
export interface SeriesAllocArgs extends BaseSeriesArgs {
  capacity: number;
  dataType: CrudeDataType;
}

const FULL_BUFFER = -1;

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
    ).buffer,
);

const nullArrayZ = z
  .union([z.null(), z.undefined()])
  .transform(() => new Uint8Array().buffer);

const NEW_LINE = 10;

type JSType = "string" | "number" | "bigint";

const checkAsType = (jsType: JSType, dataType: DataType) => {
  if (jsType === "number" && !dataType.isNumeric)
    throw new Error(`cannot convert series of type ${dataType.toString()} to number`);
  if (jsType === "bigint" && !dataType.usesBigInt)
    throw new Error(`cannot convert series of type ${dataType.toString()} to bigint`);
};

const SERIES_DISCRIMINATOR = "sy_x_telem_series";

/**
 * Series is a strongly typed array of telemetry samples backed by an underlying binary
 * buffer.
 */
export class Series<T extends TelemValue = TelemValue>
  implements instance.Discriminated
{
  /**
   * A unique identifier for the series. If specified by the user, it is their
   * responsibility to ensure that it is unique. If not specified, a new ID will be
   * generated.
   */
  key: string;
  /**
   * A discriminator used for identifying instances of the series class even
   * when bundlers mangle the class name.
   */
  discriminator: string = SERIES_DISCRIMINATOR;
  /** The data type of the series. */
  readonly dataType: DataType;
  /**
   * A sample offset that can be used to shift the values of all samples upwards or
   * downwards. Useful to convert series to lower precision data types while preserving
   * the relative range of actual values.
   */
  sampleOffset: math.Numeric;
  /**
   * Stores information about the buffer state of this array into a WebGL buffer.
   */
  private readonly gl: GL;
  /** The underlying data. */
  private readonly _data: ArrayBuffer;
  /** The time range occupied by the series' data. */
  readonly timeRange: TimeRange = TimeRange.ZERO;
  /**
   * Alignment defines the location of the series relative to other series in a logical
   * group. Useful for defining the position of the series within a channel's data.
   */
  readonly alignment: bigint = 0n;
  /**
   * Alignment multiple defines the number of alignment steps taken per sample. This is
   * useful for when the samples in a series represent a partial view of the raw data
   * i.e. decimation or averaging.
   */
  readonly alignmentMultiple: bigint = 1n;
  /** A cached minimum value. */
  private cachedMin?: math.Numeric;
  /** A cached maximum value. */
  private cachedMax?: math.Numeric;
  /** The write position of the buffer. */
  private writePos: number = FULL_BUFFER;
  /** Tracks the number of entities currently using this array. */
  private _refCount: number = 0;
  /** Caches the length of the array for variable length data types. */
  private cachedLength?: number;
  /** Caches the indexes of the array for variable length data types. */
  private _cachedIndexes?: number[];

  /**
   * A zod schema that can be used to validate that a particular value
   * can be constructed into a series.
   */
  static readonly crudeZ = z.object({
    timeRange: TimeRange.z.optional(),
    dataType: DataType.z,
    alignment: z.coerce.bigint().optional(),
    data: z.union([
      stringArrayZ,
      nullArrayZ,
      z.instanceof(ArrayBuffer),
      z.instanceof(Uint8Array),
    ]),
    glBufferUsage: glBufferUsageZ.default("static").optional(),
  });

  /**
   * A zod schema that validates and constructs a series from it's crude
   * representation.
   */
  static readonly z = Series.crudeZ.transform((props) => new Series(props));
  /**
   * The Series constructor accepts either a SeriesArgs object or a CrudeSeries value.
   *
   * SeriesArgs interface properties:
   * @property {CrudeSeries | null} [data] - The data to construct the series from. Can be:
   *   - A typed array (e.g. Float32Array, Int32Array)
   *   - A JS array of numbers, strings, or objects
   *   - A single value (number, string, bigint, etc.)
   *   - An ArrayBuffer
   *   - Another Series instance
   * @property {CrudeDataType} [dataType] - The data type of the series. If not provided,
   *   will be inferred from the data. Required when constructing from an ArrayBuffer, or
   *   an empty JS array.
   * @property {TimeRange} [timeRange] - The time range occupied by the series' data.
   *   Defaults to TimeRange.ZERO.
   * @property {math.Numeric} [sampleOffset] - An offset to apply to each sample value.
   *   Useful for converting arrays to lower precision while preserving relative range.
   *   Defaults to 0.
   * @property {GLBufferUsage} [glBufferUsage] - The WebGL buffer usage hint. Can be
   *   "static" or "dynamic". Defaults to "static".
   * @property {bigint} [alignment] - The logical position of the series relative to other
   *   series in a group. Defaults to 0n.
   * @property {string} [key] - A unique identifier for the series. If not provided,
   *   a new ID will be generated.
   *
   * @example
   * // Create a series from a typed array
   * const s1 = new Series(new Float32Array([1, 2, 3]));
   *
   * @example
   * // Create a series from a JS array with explicit data type
   * const s2 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
   *
   * @example
   * // Create a series from a single value (data type inferred)
   * const s3 = new Series(1); // Creates a FLOAT64 series
   * const s4 = new Series("abc"); // Creates a STRING series
   * const s5 = new Series(1n); // Creates an INT64 series
   *
   * @example
   * // Create a series from objects (automatically uses JSON data type)
   * const s6 = new Series([{ a: 1, b: "apple" }]);
   *
   * @example
   * // Create a series with time range and alignment
   * const s7 = new Series({
   *   data: new Float32Array([1, 2, 3]),
   *   timeRange: new TimeRange(1, 2),
   *   alignment: 1n
   * });
   *
   * @example
   * // Create a series from another series (copies properties)
   * const s8 = new Series(s1);
   *
   * @example
   * // Create a series with sample offset
   * const s9 = new Series({
   *   data: new Float32Array([1, 2, 3]),
   *   sampleOffset: 2
   * }); // Values will be 3, 4, 5
   *
   * @example
   * // Create a series with WebGL buffer usage
   * const s10 = new Series({
   *   data: new Float32Array([1, 2, 3]),
   *   glBufferUsage: "dynamic"
   * });
   *
   * @throws Error if constructing from an empty JS array without specifying data type
   * @throws Error if constructing from an ArrayBuffer without specifying data type
   * @throws Error if data type cannot be inferred from input
   */
  constructor(props: SeriesArgs | CrudeSeries) {
    if (isCrudeSeries(props)) props = { data: props };
    props.data ??= [];
    const {
      dataType,
      timeRange,
      sampleOffset = 0,
      glBufferUsage = "static",
      alignment = 0n,
      alignmentMultiple = 1n,
      key = id.create(),
      data,
    } = props;
    if (isSeries(data)) {
      const data_ = data as Series;
      this.key = data_.key;
      this.dataType = data_.dataType;
      this.sampleOffset = data_.sampleOffset;
      this.gl = data_.gl;
      this._data = data_._data;
      this.timeRange = data_.timeRange;
      this.alignment = data_.alignment;
      this.alignmentMultiple = data_.alignmentMultiple;
      this.cachedMin = data_.cachedMin;
      this.cachedMax = data_.cachedMax;
      this.writePos = data_.writePos;
      this._refCount = data_._refCount;
      this.cachedLength = data_.cachedLength;
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
    else if (isArray && data.length === 0)
      this._data = new this.dataType.Array([]).buffer;
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
        this.cachedLength = data_.length;
        this._data = new TextEncoder().encode(`${data_.join("\n")}\n`).buffer;
      } else if (this.dataType.equals(DataType.JSON)) {
        this.cachedLength = data_.length;
        this._data = new TextEncoder().encode(
          `${data_.map((d) => binary.JSON_CODEC.encodeString(d)).join("\n")}\n`,
        ).buffer;
      } else if (this.dataType.usesBigInt && typeof first === "number")
        this._data = new this.dataType.Array(
          data_.map((v) => BigInt(Math.round(v as number))),
        ).buffer;
      else if (!this.dataType.usesBigInt && typeof first === "bigint")
        this._data = new this.dataType.Array(
          data_.map(Number) as number[] & bigint[],
        ).buffer;
      else this._data = new this.dataType.Array(data_ as number[] & bigint[]).buffer;
    }

    this.key = key;
    this.alignment = alignment;
    this.alignmentMultiple = alignmentMultiple;
    this.sampleOffset = sampleOffset ?? 0;
    this.timeRange = timeRange ?? TimeRange.ZERO;
    this.gl = {
      control: null,
      buffer: null,
      prevBuffer: 0,
      bufferUsage: glBufferUsage,
    };
  }

  /**
   * Allocates a new series with a given capacity and data type.
   * @param args.capacity the capacity of the series in samples. If the data type is of
   * variable density (i.e. JSON, STRING, BYTES), this is the capacity in bytes.
   * @param args.dataType the data type of the series.
   * @param args.rest the rest of the arguments to pass to the series constructor.
   */
  static alloc({ capacity, dataType, ...rest }: SeriesAllocArgs): Series {
    if (capacity === 0)
      throw new Error("[Series] - cannot allocate an array of length 0");
    const data = new new DataType(dataType).Array(capacity);
    const arr = new Series({ data: data.buffer, dataType, ...rest });
    arr.writePos = 0;
    return arr;
  }

  /**
   * @returns the number of references to this series i.e. the number of times this
   * series has been acquired (by calling acquire) and not released (by calling
   * release).
   */
  get refCount(): number {
    return this._refCount;
  }

  /**
   * Acquires a reference to this series, optionally buffering its data into the
   * specified buffer controller. This method is useful for managing the life span
   * of series buffered to the GPU.
   * @param gl the buffer controller to buffer the series to. If not provided, the series
   * will not be buffered to the GPU.
   */
  acquire(gl?: GLBufferController): void {
    this._refCount++;
    if (gl != null) this.updateGLBuffer(gl);
  }

  /**
   * Releases a reference to this series. If the reference count to the series reaches
   * 0 and the series has been buffered to the GPU, the series will be deleted from
   * the GPU.
   */
  release(): void {
    this._refCount--;
    if (this.refCount === 0 && this.gl.control != null)
      this.maybeGarbageCollectGLBuffer(this.gl.control);
    else if (this.refCount < 0)
      console.warn("cannot release a series with a negative reference count");
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
    if (this.cachedLength != null) {
      this.cachedLength += toWrite.length;
      this.calculateCachedLength();
    }
    return toWrite.length;
  }

  private writeFixed(other: Series): number {
    if (this.writePos === FULL_BUFFER) return 0;
    const available = this.capacity - this.writePos;
    const toWrite = other.sub(0, available);
    this.writeToUnderlyingData(toWrite);
    this.cachedLength = undefined;
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

  /**
   * Returns a native JS typed array with the proper data type.
   * If the series is not full, returns a view of the data up to the write position.
   * @returns A typed array containing the series data.
   */
  get data(): TypedArray {
    if (this.writePos === FULL_BUFFER) return this.underlyingData;
    return new this.dataType.Array(this._data, 0, this.writePos);
  }

  /**
   * Returns an array of the values in the series as strings.
   * For variable length data types (like STRING or JSON), this decodes the underlying buffer.
   * @returns An array of string representations of the series values.
   */
  toStrings(): string[] {
    if (this.dataType.isVariable)
      return new TextDecoder().decode(this.underlyingData).split("\n").slice(0, -1);
    return Array.from(this).map((d) => d.toString());
  }

  /**
   * Parses a JSON series into an array of values using the provided zod schema.
   * @template Z The zod schema type.
   * @param schema The zod schema to use to parse the JSON series.
   * @throws Error if the series does not have a data type of JSON.
   * @returns An array of values parsed from the JSON series.
   */
  parseJSON<Z extends z.ZodType>(schema: Z): Array<z.infer<Z>> {
    if (!this.dataType.equals(DataType.JSON))
      throw new Error("cannot parse non-JSON series as JSON");
    return this.toStrings().map((s) => schema.parse(binary.JSON_CODEC.decodeString(s)));
  }

  /**
   * Returns the capacity of the series in bytes.
   * @returns The size of the underlying buffer in bytes.
   */
  get byteCapacity(): Size {
    return new Size(this.underlyingData.byteLength);
  }

  /**
   * Returns the capacity of the series in samples.
   * For variable length data types, this is the capacity in bytes.
   * @returns The number of samples that can be stored in the series.
   */
  get capacity(): number {
    if (this.dataType.isVariable) return this.byteCapacity.valueOf();
    return this.dataType.density.length(this.byteCapacity);
  }

  /**
   * Returns the length of the series in bytes.
   * For variable length data types, this is the actual number of bytes used.
   * @returns The size of the data in bytes.
   */
  get byteLength(): Size {
    if (this.writePos === FULL_BUFFER) return this.byteCapacity;
    if (this.dataType.isVariable) return new Size(this.writePos);
    return this.dataType.density.size(this.writePos);
  }

  /**
   * Returns the number of samples in this array.
   * For variable length data types, this is calculated by counting newlines.
   * @returns The number of samples in the series.
   */
  get length(): number {
    if (this.cachedLength != null) return this.cachedLength;
    if (this.dataType.isVariable) return this.calculateCachedLength();
    if (this.writePos === FULL_BUFFER)
      return this.byteCapacity.valueOf() / this.dataType.density.valueOf();
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
    this.cachedLength = cl;
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
      timeRange: this.timeRange,
      sampleOffset,
      glBufferUsage: this.gl.bufferUsage,
      alignment: this.alignment,
    });
  }

  private calcRawMax(): math.Numeric {
    if (this.length === 0) return -Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP))
      this.cachedMax = this.data[this.data.length - 1];
    else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this.cachedMax = d.reduce((a, b) => (a > b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this.cachedMax = d.reduce((a, b) => (a > b ? a : b));
    }
    return this.cachedMax;
  }

  /** @returns the maximum value in the array */
  get max(): math.Numeric {
    return this.calcMax();
  }

  private calcMax(): math.Numeric {
    if (this.dataType.isVariable)
      throw new Error("cannot calculate maximum on a variable length data type");
    if (this.writePos === 0) return -Infinity;
    this.cachedMax ??= this.calcRawMax();
    return math.add(this.cachedMax, this.sampleOffset);
  }

  private calcRawMin(): math.Numeric {
    if (this.length === 0) return Infinity;
    if (this.dataType.equals(DataType.TIMESTAMP)) this.cachedMin = this.data[0];
    else if (this.dataType.usesBigInt) {
      const d = this.data as BigInt64Array;
      this.cachedMin = d.reduce((a, b) => (a < b ? a : b));
    } else {
      const d = this.data as Float64Array;
      this.cachedMin = d.reduce((a, b) => (a < b ? a : b));
    }
    return this.cachedMin;
  }

  /** @returns the minimum value in the array */
  get min(): math.Numeric {
    return this.calcMin();
  }

  private calcMin(): math.Numeric {
    if (this.dataType.isVariable)
      throw new Error("cannot calculate minimum on a variable length data type");
    if (this.writePos === 0) return Infinity;
    this.cachedMin ??= this.calcRawMin();
    return math.add(this.cachedMin, this.sampleOffset);
  }

  /** @returns the bounds of the series. */
  get bounds(): bounds.Bounds {
    return bounds.construct(Number(this.min), Number(this.max), { makeValid: false });
  }

  private maybeRecomputeMinMax(update: Series): void {
    if (this.cachedMin != null) {
      const min = update.cachedMin ?? update.calcRawMin();
      if (min < this.cachedMin) this.cachedMin = min;
    }
    if (this.cachedMax != null) {
      const max = update.cachedMax ?? update.calcRawMax();
      if (max > this.cachedMax) this.cachedMax = max;
    }
  }

  /**
   * @returns the value at the given alignment.
   * @param alignment the alignment to get the value at.
   * @param required throws an error if the value is not found.
   */
  atAlignment(alignment: bigint, required: true): T;

  /**
   * @returns the value at the given alignment.
   * @param alignment the alignment to get the value at.
   * @param required throws an error if the value is not found.
   */
  atAlignment(alignment: bigint, required?: false): T | undefined;

  atAlignment(alignment: bigint, required?: boolean): T | undefined {
    const index = Number((alignment - this.alignment) / this.alignmentMultiple);
    if (index < 0 || index >= this.length) {
      if (required === true) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
    return this.at(index, required as true);
  }

  /**
   * @returns the value at the given index.
   * @param index the index to get the value at.
   * @param required throws an error if the value is not found.
   */
  at(index: number, required: true): T;

  /**
   * @returns the value at the given index.
   * @param index the index to get the value at.
   * @param required throws an error if the value is not found.
   */
  at(index: number, required?: false): T | undefined;

  at(index: number, required: boolean = false): T | undefined {
    if (this.dataType.isVariable) return this.atVariable(index, required ?? false);
    if (this.dataType.equals(DataType.UUID)) return this.atUUID(index, required) as T;
    if (index < 0) index = this.length + index;
    const v = this.data[index];
    if (v == null) {
      if (required === true) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
    return math.add(v, this.sampleOffset) as T;
  }

  private atUUID(index: number, required: boolean): string | undefined {
    if (index < 0) index = this.length + index;
    const uuidString = uuid.parse(
      new Uint8Array(this.buffer, index * this.dataType.density.valueOf()),
    );
    if (uuidString == null) {
      if (required) throw new Error(`[series] - no value at index ${index}`);
      return undefined;
    }
    return uuidString;
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

  /**
   * Updates the WebGL buffer for the series if it is not up to date. This method
   * should be called whenever a series has been previously buffered to the GPU and
   * then modified via calls to write().
   * @param gl the buffer controller to update the buffer for. This controller should
   * be the same buffer previously passed to {@method acquire} or {@method updateGLBuffer}.
   */
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

  /**
   * Reinterprets the series as containing strings as its JS primitive type.
   * @throws if the series does not have a data type of STRING or JSON.
   */
  as(jsType: "string"): Series<string>;

  /**
   * Reinterprets the series as containing numbers as its JS primitive type.
   * @throws if the series does not have a numeric data type.
   */
  as(jsType: "number"): Series<number>;

  /**
   * Reinterprets the series as containing bigints as its JS primitive type.
   * @throws if the series does not have a data type that requires bigints i.e.
   * INT64 and UINT64.
   */
  as(jsType: "bigint"): Series<bigint>;

  as<T extends TelemValue>(jsType: "string" | "number" | "bigint"): Series<T> {
    checkAsType(jsType, this.dataType);
    return this as unknown as Series<T>;
  }

  /** @returns a digest containing information about the series. */
  get digest(): SeriesDigest {
    return {
      key: this.key,
      dataType: this.dataType.toString(),
      sampleOffset: this.sampleOffset,
      alignment: {
        lower: alignmentDigest(this.alignmentBounds.lower),
        upper: alignmentDigest(this.alignmentBounds.upper),
        multiple: this.alignmentMultiple,
      },
      timeRange: this.timeRange.toString(),
      length: this.length,
      capacity: this.capacity,
    };
  }

  /**
   * @returns the alignment bounds of the series, representing the logical space
   * occupied by the series in a group of series. This is typically used to order the
   * series within a channel's data.
   *
   * The lower bound is the alignment of the first sample, and the upper bound is the
   * alignment of the last sample + 1. The lower bound is inclusive, while the upper bound
   * is exclusive.
   */
  get alignmentBounds(): bounds.Bounds<bigint> {
    return bounds.construct(
      this.alignment,
      this.alignment + BigInt(this.length) * this.alignmentMultiple,
    );
  }

  private maybeGarbageCollectGLBuffer(gl: GLBufferController): void {
    if (this.gl.buffer == null) return;
    gl.deleteBuffer(this.gl.buffer);
    this.gl.buffer = null;
    this.gl.prevBuffer = 0;
    this.gl.control = null;
  }

  /**
   * @returns the WebGL buffer for the series. This method should only be called after
   * the series has been buffered to the GPU via a call to {@method acquire} or
   * {@method updateGLBuffer}.
   * @throws if the series has not been buffered to the GPU.
   */
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
    if (this.dataType.equals(DataType.UUID))
      return new UUIDSeriesIterator(this) as Iterator<T>;

    return new FixedSeriesIterator(this) as Iterator<T>;
  }

  /**
   * Returns a slice of the series from start to end.
   * @param start The start index (inclusive).
   * @param end The end index (exclusive).
   * @returns A new series containing the sliced data.
   */
  slice(start: number, end?: number): Series {
    return this.sliceSub(false, start, end);
  }

  /**
   * Returns a subarray view of the series from start to end.
   * @param start The start index (inclusive).
   * @param end The end index (exclusive).
   * @returns A new series containing the subarray data.
   */
  sub(start: number, end?: number): Series {
    return this.sliceSub(true, start, end);
  }

  /**
   * Returns an iterator over a portion of the series.
   * @param start The start index (inclusive).
   * @param end The end index (exclusive).
   * @returns An iterator over the specified range.
   */
  subIterator(start: number, end?: number): Iterator<T> {
    return new SubIterator(this, start, end ?? this.length);
  }

  /**
   * Returns an iterator over a portion of the series based on alignment.
   * @param start The start alignment (inclusive).
   * @param end The end alignment (exclusive).
   * @returns An iterator over the specified alignment range.
   */
  subAlignmentIterator(start: bigint, end: bigint): Iterator<T> {
    const startIdx = Math.ceil(
      Number(start - this.alignment) / Number(this.alignmentMultiple),
    );
    const endIdx = Math.ceil(
      Number(end - this.alignment) / Number(this.alignmentMultiple),
    );
    return new SubIterator(this, startIdx, endIdx);
  }

  private subBytes(start: number, end?: number): Series {
    if (start >= 0 && (end == null || end >= this.byteLength.valueOf())) return this;
    const data = this.data.subarray(start, end);
    return new Series({
      data,
      dataType: this.dataType,
      timeRange: this.timeRange,
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
      timeRange: this.timeRange,
      sampleOffset: this.sampleOffset,
      glBufferUsage: this.gl.bufferUsage,
      alignment: this.alignment + BigInt(start),
    });
  }

  /**
   * Creates a new series with a different alignment.
   * @param alignment The new alignment value.
   * @returns A new series with the specified alignment.
   */
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

  /**
   * Returns a string representation of the series.
   * For series with more than 10 elements, shows the first 5 and last 5 elements.
   * @returns A string representation of the series.
   */
  toString(): string {
    let data = `Series(${this.dataType.toString()} ${this.length} [`;
    if (this.length <= 10) data += Array.from(this).map((v) => v.toString());
    else {
      for (let i = 0; i < 5; i++) {
        data += `${this.at(i)?.toString()}`;
        data += ",";
      }
      data += "...,";
      for (let i = -5; i < 0; i++) {
        data += this.at(i)?.toString();
        if (i < -1) data += ",";
      }
    }
    data += "])";
    return data;
  }
}

/** @returns true if a Series can be constructed from the given value, and false otherwise. */
export const isCrudeSeries = (value: unknown): value is CrudeSeries => {
  if (value == null) return false;
  if (Array.isArray(value)) return true;
  if (value instanceof ArrayBuffer) return true;
  if (ArrayBuffer.isView(value) && !(value instanceof DataView)) return true;
  if (value instanceof Series) return true;
  return isTelemValue(value);
};

/** @returns true if the given value is an instance of the series class. */
const isSeries = instance.createMatcher<Series>(SERIES_DISCRIMINATOR, Series);

class SubIterator<T> implements Iterator<T> {
  private readonly series: Series;
  private readonly end: number;
  private index: number;

  constructor(series: Series, start: number, end: number) {
    this.series = series;
    const b = bounds.construct(0, series.length + 1);
    this.end = bounds.clamp(b, end);
    this.index = bounds.clamp(b, start);
  }

  next(): IteratorResult<T> {
    if (this.index >= this.end) return { done: true, value: undefined };
    return { done: false, value: this.series.at(this.index++, true) as T };
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
}

class JSONSeriesIterator implements Iterator<unknown> {
  private readonly wrapped: Iterator<string>;
  private static schema = z.record(z.string(), z.unknown());

  constructor(wrapped: Iterator<string>) {
    this.wrapped = wrapped;
  }

  next(): IteratorResult<object> {
    const next = this.wrapped.next();
    if (next.done === true) return { done: true, value: undefined };
    return {
      done: false,
      value: binary.JSON_CODEC.decodeString(next.value, JSONSeriesIterator.schema),
    };
  }
}

class UUIDSeriesIterator implements Iterator<string> {
  private readonly series: Series;
  private index: number;
  private readonly data: Uint8Array;
  private readonly density: number;

  constructor(series: Series) {
    if (!series.dataType.equals(DataType.UUID))
      throw new Error("cannot create a UUID series iterator for a non-UUID series");
    this.series = series;
    this.index = 0;
    this.data = new Uint8Array(series.buffer);
    this.density = DataType.UUID.density.valueOf();
  }

  next(): IteratorResult<string> {
    if (this.index >= this.series.length) return { done: true, value: undefined };
    const uuidString = uuid.parse(this.data, this.index * this.density);
    this.index++;
    return { done: false, value: uuidString };
  }
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
}

/**
 * MultiSeries represents a collection of Series instances that share the same data type.
 * It provides a unified interface for working with multiple series as if they were a single
 * continuous series.
 *

 */
export class MultiSeries<T extends TelemValue = TelemValue> implements Iterable<T> {
  /** The array of series in this collection */
  readonly series: Array<Series<T>>;

  /**
   * The MultiSeries constructor accepts an optional array of Series instances. All series
   * in the collection must have the same data type.
   *
   * @example
   * // Create an empty MultiSeries
   * const ms1 = new MultiSeries();
   *
   * @example
   * // Create a MultiSeries from multiple numeric series
   * const s1 = new Series(new Float32Array([1, 2, 3]));
   * const s2 = new Series(new Float32Array([4, 5, 6]));
   * const ms2 = new MultiSeries([s1, s2]);
   *
   * @example
   * // Create a MultiSeries from string series
   * const s3 = new Series(["apple", "banana"]);
   * const s4 = new Series(["carrot", "date"]);
   * const ms3 = new MultiSeries([s3, s4]);
   *
   * @example
   * // Create a MultiSeries from JSON series
   * const s5 = new Series([{ a: 1, b: "apple" }]);
   * const s6 = new Series([{ a: 2, b: "banana" }]);
   * const ms4 = new MultiSeries([s5, s6]);
   *
   * @example
   * // Add series to an existing MultiSeries
   * const ms5 = new MultiSeries();
   * ms5.push(s1);
   * ms5.push(s2);
   *
   * @example
   * // Combine two MultiSeries
   * const ms6 = new MultiSeries([s1]);
   * const ms7 = new MultiSeries([s2]);
   * ms6.push(ms7);
   *
   * @throws Error if attempting to add a series with a different data type
   */
  constructor(series: Array<Series<T>> = []) {
    if (series.length !== 0) {
      const type = series[0].dataType;
      for (let i = 1; i < series.length; i++)
        if (!series[i].dataType.equals(type))
          throw new Error("[multi-series] - series must have the same data type");
    }
    this.series = series;
  }

  /**
   * Reinterprets the series as containing strings as its JS primitive type.
   * @throws if the series does not have a data type of STRING or JSON.
   */
  as(jsType: "string"): MultiSeries<string>;

  /**
   * Reinterprets the series as containing numbers as its JS primitive type.
   * @throws if the series does not have a numeric data type.
   */
  as(jsType: "number"): MultiSeries<number>;

  /**
   * Reinterprets the series as containing bigints as its JS primitive type.
   * @throws if the series does not have a data type that requires bigints i.e.
   * INT64 and UINT64.
   */
  as(jsType: "bigint"): MultiSeries<bigint>;

  as<T extends TelemValue>(jsType: "string" | "number" | "bigint"): MultiSeries<T> {
    checkAsType(jsType, this.dataType);
    return this as unknown as MultiSeries<T>;
  }

  /**
   * Returns the data type of the series in this collection. If the collection is empty,
   * returns DataType.UNKNOWN.
   */
  get dataType(): DataType {
    if (this.series.length === 0) return DataType.UNKNOWN;
    return this.series[0].dataType;
  }

  /**
   * Returns the combined time range of all series in the collection. If the collection
   * is empty, returns TimeRange.ZERO. The time range spans from the start of the first
   * series to the end of the last series.
   */
  get timeRange(): TimeRange {
    if (this.series.length === 0) return TimeRange.ZERO;
    return new TimeRange(
      this.series[0].timeRange.start,
      this.series[this.series.length - 1].timeRange.end,
    );
  }

  /**
   * Returns the alignment of the first series in the collection. If the collection is
   * empty, returns 0n.
   */
  get alignment(): bigint {
    if (this.series.length === 0) return 0n;
    return this.series[0].alignment;
  }

  /**
   * Returns the alignment bounds of the entire collection. The lower bound is the
   * alignment of the first series, and the upper bound is the alignment of the last
   * series + its length. If the collection is empty, returns bounds.construct(0n, 0n).
   */
  get alignmentBounds(): bounds.Bounds<bigint> {
    if (this.series.length === 0) return bounds.construct(0n, 0n);
    return bounds.construct(
      this.series[0].alignmentBounds.lower,
      this.series[this.series.length - 1].alignmentBounds.upper,
    );
  }

  /**
   * Adds a series or another MultiSeries to this collection.
   * @param series - The series or MultiSeries to add. Must have the same data type
   * as the existing series in this collection.
   * @throws Error if the series being added has a different data type
   */
  push(series: Series<T>): void;
  push(series: MultiSeries<T>): void;

  push(series: Series<T> | MultiSeries<T>): void {
    const invalidDataTypeError = () =>
      new Error(
        `cannot push a ${series.dataType.toString()} series to a ${this.dataType.toString()} multi-series`,
      );
    const dtsEqual = series.dataType.equals(this.dataType);
    if (isSeries(series)) {
      if (this.series.length !== 0 && !dtsEqual) throw invalidDataTypeError();
      this.series.push(series);
    } else {
      if (this.series.length !== 0 && series.series.length !== 0 && !dtsEqual)
        throw invalidDataTypeError();
      this.series.push(...series.series);
    }
  }

  /**
   * Returns the total length of all series in the collection.
   * @returns The sum of the lengths of all series.
   */
  get length(): number {
    return this.series.reduce((a, b) => a + b.length, 0);
  }

  /**
   * Returns the value at the specified alignment.
   * @param alignment - The alignment to get the value at.
   * @param required - If true, throws an error if the value is not found.
   * @returns The value at the specified alignment, or undefined if not found.
   * @throws Error if required is true and the value is not found.
   */
  atAlignment(alignment: bigint, required: true): T;
  atAlignment(alignment: bigint, required?: false): T | undefined;
  atAlignment(alignment: bigint, required?: boolean): T | undefined {
    for (const ser of this.series)
      if (bounds.contains(ser.alignmentBounds, alignment))
        return ser.atAlignment(alignment, required as true);
    if (required) throw new Error(`[series] - no value at alignment ${alignment}`);
    return undefined;
  }

  /**
   * Returns the value at the specified index.
   * @param index - The index to get the value at.
   * @param required - If true, throws an error if the value is not found.
   * @returns The value at the specified index, or undefined if not found.
   * @throws Error if required is true and the value is not found.
   */
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

  /**
   * Returns an iterator over a portion of the multi-series.
   * @param start - The start index (inclusive).
   * @param end - The end index (exclusive).
   * @returns An iterator over the specified range.
   */
  subIterator(start: number, end?: number): IterableIterator<T> {
    return new MultiSubIterator(this, start, end ?? this.length);
  }

  /**
   * Returns an iterator over a portion of the multi-series based on alignment.
   * @param start - The start alignment (inclusive).
   * @param end - The end alignment (exclusive).
   * @returns An iterator over the specified alignment range.
   */
  subAlignmentIterator(start: bigint, end: bigint): IterableIterator<T> {
    if (start >= this.alignmentBounds.upper || end <= this.alignmentBounds.lower)
      return noopIterableIterator;
    let startIdx = 0;
    for (let i = 0; i < this.series.length; i++) {
      const ser = this.series[i];
      if (start < ser.alignment) break;
      else if (start >= ser.alignmentBounds.upper) startIdx += ser.length;
      else if (bounds.contains(ser.alignmentBounds, start)) {
        startIdx += Math.ceil(
          Number(start - ser.alignment) / Number(ser.alignmentMultiple),
        );
        break;
      }
    }
    let endIdx = 0;
    for (let i = 0; i < this.series.length; i++) {
      const ser = this.series[i];
      if (end < ser.alignment) break;
      else if (end >= ser.alignmentBounds.upper) endIdx += ser.length;
      else if (bounds.contains(ser.alignmentBounds, end)) {
        endIdx += Math.ceil(
          Number(end - ser.alignment) / Number(ser.alignmentMultiple),
        );
        break;
      }
    }
    return new MultiSubIterator(this, startIdx, endIdx);
  }

  /**
   * Returns an iterator over the specified alignment range and span.
   * @param start - The start alignment (inclusive).
   * @param span - The number of samples to include.
   * @returns An iterator over the specified range.
   */
  subAlignmentSpanIterator(start: bigint, span: number): IterableIterator<T> {
    if (start >= this.alignmentBounds.upper) return noopIterableIterator;
    span = Math.min(span, Number(this.distance(start, this.alignmentBounds.upper)));
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

  /**
   * Updates the WebGL buffer for all series in the collection.
   * @param gl - The WebGL buffer controller to use.
   */
  updateGLBuffer(gl: GLBufferController): void {
    this.series.forEach((s) => s.updateGLBuffer(gl));
  }

  /**
   * Returns the bounds containing the minimum and maximum values across all series.
   */
  get bounds(): bounds.Bounds {
    return bounds.max(this.series.map((s) => s.bounds));
  }

  /**
   * Returns the sum of the byte lengths of all series.
   */
  get byteLength(): Size {
    return new Size(this.series.reduce((a, b) => a + b.byteLength.valueOf(), 0));
  }

  /**
   * Returns a combined typed array containing all data from all series.
   * @returns A typed array containing all data from all series.
   */
  get data(): TypedArray {
    const buf = new this.dataType.Array(this.length);
    let offset = 0;
    for (const ser of this.series) {
      buf.set(ser.data as ArrayLike<any>, offset);
      offset += ser.length;
    }
    return new this.dataType.Array(buf.buffer);
  }

  /**
   * Traverses the alignment space by a given distance from a start point.
   * @param start - The starting alignment.
   * @param dist - The distance to traverse.
   * @returns The resulting alignment after traversal.
   */
  traverseAlignment(start: bigint, dist: bigint): bigint {
    const b = this.series.map((s) => s.alignmentBounds);
    return bounds.traverse(b, start, dist);
  }

  /**
   * Acquires a reference to the WebGL buffer for all series.
   * @param gl - Optional WebGL buffer controller to use.
   */
  acquire(gl?: GLBufferController): void {
    this.series.forEach((s) => s.acquire(gl));
  }

  /**
   * Releases the WebGL buffer reference for all series.
   */
  release(): void {
    this.series.forEach((s) => s.release());
  }

  /**
   * Calculates the number of samples between two alignments in the multi-series.
   * @param start - The starting alignment.
   * @param end - The ending alignment.
   * @returns The distance between the alignments.
   */
  distance(start: bigint, end: bigint): bigint {
    const b = this.series.map((s) => s.alignmentBounds);
    return bounds.distance(b, start, end);
  }

  /**
   * Parses a JSON multi-series into an array of values using the provided zod schema.
   * @template Z - The zod schema type.
   * @param schema - The zod schema to use to parse the JSON series.
   * @throws Error if the series does not have a data type of JSON.
   * @returns An array of values parsed from the JSON series.
   */
  parseJSON<Z extends z.ZodType>(schema: Z): Array<z.infer<Z>> {
    if (!this.dataType.equals(DataType.JSON))
      throw new Error("cannot parse non-JSON series as JSON");
    return this.series.flatMap((s) => s.parseJSON(schema));
  }

  /**
   * Returns an iterator over all values in the multi-series.
   * @returns An iterator that yields all values from all series in sequence.
   */
  [Symbol.iterator](): Iterator<T> {
    if (this.series.length === 0)
      return {
        next(): IteratorResult<T> {
          return { done: true, value: undefined };
        },
      };
    return new MultiSeriesIterator<T>(this.series);
  }

  /**
   * Returns an array of the values in the multi-series as strings.
   * For variable length data types (like STRING or JSON), this decodes the underlying buffer.
   * @returns An array of string representations of the multi-series values.
   */
  toStrings(): string[] {
    return this.series.flatMap((s) => s.toStrings());
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
}

class MultiSubIterator<
  T extends TelemValue = TelemValue,
> implements IterableIterator<T> {
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
    return { done: false, value: this.series.at(this.index++, true) };
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
