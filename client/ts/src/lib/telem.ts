// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { registerCustomTypeEncoder } from "@synnaxlabs/freighter";

const valueOfEncoder = (value: unknown): unknown => value?.valueOf();

/** Represents a nanosecond precision UTC timestamp. */
export class TimeStamp extends Number {
  constructor(value: UnparsedTimeStamp) {
    if (value instanceof Number) super(value.valueOf());
    else super(value);
  }

  /**
   * @returns A JavaScript Date object representing the TimeStamp.
   */
  date(): Date {
    return new Date(this.milliseconds());
  }

  /**
   * Checks if the TimeStamp is equal to another TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamps are equal, false otherwise.
   */
  equals(other: UnparsedTimeStamp): boolean {
    return this.valueOf() === new TimeStamp(other).valueOf();
  }

  /**
   * Creates a TimeSpan representing the duration between the two timestamps.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns A TimeSpan representing the duration between the two timestamps.
   *   The span is guaranteed to be positive.
   */
  span(other: UnparsedTimeStamp): TimeSpan {
    return this.range(other).span();
  }

  /**
   * Creates a TimeRange spanning the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns A TimeRange spanning the given TimeStamp that is guaranteed to be
   *   valid, regardless of the TimeStamp order.
   */
  range(other: UnparsedTimeStamp): TimeRange {
    return new TimeRange(this, other).makeValid();
  }

  /**
   * Creates a TimeRange starting at the TimeStamp and spanning the given
   * TimeSpan.
   *
   * @param other - The TimeSpan to span.
   * @returns A TimeRange starting at the TimeStamp and spanning the given
   *   TimeSpan. The TimeRange is guaranteed to be valid.
   */
  spanRange(other: UnparsedTimeSpan): TimeRange {
    return this.range(this.add(other)).makeValid();
  }

  /**
   * Checks if the TimeStamp represents the unix epoch.
   *
   * @returns True if the TimeStamp represents the unix epoch, false otherwise.
   */
  isZero(): boolean {
    return this.valueOf() === 0;
  }

  /**
   * Checks if the TimeStamp is after the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamp is after the given TimeStamp, false
   *   otherwise.
   */
  after(other: UnparsedTimeStamp): boolean {
    return this.valueOf() > new TimeStamp(other).valueOf();
  }

  /**
   * Checks if the TimeStamp is after or equal to the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamp is after or equal to the given TimeStamp,
   *   false otherwise.
   */
  afterEq(other: UnparsedTimeStamp): boolean {
    return this.valueOf() >= new TimeStamp(other).valueOf();
  }

  /**
   * Checks if the TimeStamp is before the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamp is before the given TimeStamp, false
   *   otherwise.
   */
  before(other: UnparsedTimeStamp): boolean {
    return this.valueOf() < new TimeStamp(other).valueOf();
  }

  /**
   * Checks if TimeStamp is before or equal to the current timestamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if TimeStamp is before or equal to the current timestamp,
   *   false otherwise.
   */
  beforeEq(other: UnparsedTimeStamp): boolean {
    return this.valueOf() <= new TimeStamp(other).valueOf();
  }

  /**
   * Adds a TimeSpan to the TimeStamp.
   *
   * @param span - The TimeSpan to add.
   * @returns A new TimeStamp representing the sum of the TimeStamp and
   *   TimeSpan.
   */
  add(span: UnparsedTimeSpan): TimeStamp {
    return new TimeStamp(this.valueOf() + span.valueOf());
  }

  /**
   * Subtracts a TimeSpan from the TimeStamp.
   *
   * @param span - The TimeSpan to subtract.
   * @returns A new TimeStamp representing the difference of the TimeStamp and
   *   TimeSpan.
   */
  sub(span: UnparsedTimeSpan): TimeStamp {
    return new TimeStamp(this.valueOf() - span.valueOf());
  }

  /**
   * @returns The number of milliseconds since the unix epoch.
   */
  milliseconds(): number {
    return this.valueOf() / TimeStamp.Millisecond.valueOf();
  }

  /** The maximum possible value for a timestamp */
  static readonly MAX = new TimeStamp(TimeStamp.MAX_SAFE_INTEGER);

  /** The minimum possible value for a timestamp */
  static readonly MIN = new TimeStamp(TimeStamp.MIN_SAFE_INTEGER);

  /** The unix epoch */
  static readonly ZERO = new TimeStamp(0);

  /* One nanosecond after the unix epoch */
  static readonly Nanosecond = TimeStamp.Nanoseconds(1);

  /** @returns a new TimeStamp n nanoseconds after the unix epoch */
  static Nanoseconds(value: UnparsedTimeStamp): TimeStamp {
    return new TimeStamp(value);
  }

  /** One microsecond after the unix epoch */
  static readonly Microsecond = TimeStamp.Microseconds(1);

  /** @returns a new TimeStamp n microseconds after the unix epoch */
  static Microseconds(value: UnparsedTimeStamp): TimeStamp {
    return TimeStamp.Nanoseconds(value.valueOf() * 1000);
  }

  /** One millisecond after the unix epoch */
  static readonly Millisecond = TimeStamp.Milliseconds(1);

  /** @returns a new TimeStamp n milliseconds after the unix epoch */
  static Milliseconds(value: UnparsedTimeStamp): TimeStamp {
    return TimeStamp.Microseconds(value.valueOf() * 1000);
  }

  /** One second after the unix epoch */
  static readonly Second = TimeStamp.Seconds(1);

  /** @returns a new TimeStamp n seconds after the unix epoch */
  static Seconds(value: UnparsedTimeStamp): TimeStamp {
    return TimeStamp.Milliseconds(value.valueOf() * 1000);
  }

  /** One minute after the unix epoch */
  static readonly Minute = TimeStamp.Minutes(1);

  /** @returns a new TimeStamp n minutes after the unix epoch */
  static Minutes(value: UnparsedTimeStamp): TimeStamp {
    return TimeStamp.Seconds(value.valueOf() * 60);
  }
}

/** TimeSpan represents a nanosecond precision duration. */
export class TimeSpan extends Number {
  constructor(value: UnparsedTimeSpan) {
    if (value instanceof Number) super(value.valueOf());
    else super(value);
  }

  /** @returns The number of seconds in the TimeSpan. */
  seconds(): number {
    return this.valueOf() / TimeSpan.Seconds(1).valueOf();
  }

  /** @returns The number of milliseconds in the TimeSpan. */
  milliseconds(): number {
    return this.valueOf() / TimeSpan.Milliseconds(1).valueOf();
  }

  /**
   * Checks if the TimeSpan represents a zero duration.
   *
   * @returns True if the TimeSpan represents a zero duration, false otherwise.
   */
  isZero(): boolean {
    return this.valueOf() === 0;
  }

  /**
   * Checks if the TimeSpan is equal to another TimeSpan.
   *
   * @returns True if the TimeSpans are equal, false otherwise.
   */
  equals(other: UnparsedTimeSpan): boolean {
    return this.valueOf() === new TimeSpan(other).valueOf();
  }

  /**
   * Adds a TimeSpan to the TimeSpan.
   *
   * @returns A new TimeSpan representing the sum of the two TimeSpans.
   */
  add(other: UnparsedTimeSpan): TimeSpan {
    return new TimeSpan(this.valueOf() + new TimeSpan(other).valueOf());
  }

  /**
   * Creates a TimeSpan representing the duration between the two timestamps.
   *
   * @param other
   */
  sub(other: UnparsedTimeSpan): TimeSpan {
    return new TimeSpan(this.valueOf() - new TimeSpan(other).valueOf());
  }

  /**
   * Creates a TimeSpan representing the given number of nanoseconds.
   *
   * @param value - The number of nanoseconds.
   * @returns A TimeSpan representing the given number of nanoseconds.
   */
  static Nanoseconds(value: UnparsedTimeSpan = 1): TimeSpan {
    return new TimeSpan(value);
  }

  /** A nanosecond. */
  static readonly Nanosecond = TimeSpan.Nanoseconds(1);

  /**
   * Creates a TimeSpan representing the given number of microseconds.
   *
   * @param value - The number of microseconds.
   * @returns A TimeSpan representing the given number of microseconds.
   */
  static Microseconds(value: UnparsedTimeStamp = 1): TimeSpan {
    return TimeSpan.Nanoseconds(value.valueOf() * 1000);
  }

  /** A microsecond. */
  static readonly Microsecond = TimeSpan.Microseconds(1);

  /**
   * Creates a TimeSpan representing the given number of milliseconds.
   *
   * @param value - The number of milliseconds.
   * @returns A TimeSpan representing the given number of milliseconds.
   */
  static Milliseconds(value: UnparsedTimeStamp = 1): TimeSpan {
    return TimeSpan.Microseconds(value.valueOf() * 1000);
  }

  /** A millisecond. */
  static readonly Millisecond = TimeSpan.Milliseconds(1);

  /**
   * Creates a TimeSpan representing the given number of seconds.
   *
   * @param value - The number of seconds.
   * @returns A TimeSpan representing the given number of seconds.
   */
  static Seconds(value: UnparsedTimeStamp = 1): TimeSpan {
    return TimeSpan.Milliseconds(value.valueOf() * 1000);
  }

  /** A second. */
  static readonly Second = TimeSpan.Seconds(1);

  /**
   * Creates a TimeSpan representing the given number of minutes.
   *
   * @param value - The number of minutes.
   * @returns A TimeSpan representing the given number of minutes.
   */
  static Minutes(value: UnparsedTimeStamp = 1): TimeSpan {
    return TimeSpan.Seconds(value.valueOf() * 60);
  }

  /** A minute. */
  static readonly Minute = TimeSpan.Minutes(1);

  /**
   * Creates a TimeSpan representing the given number of hours.
   *
   * @param value - The number of hours.
   * @returns A TimeSpan representing the given number of hours.
   */
  static Hours(value: UnparsedTimeStamp = 1): TimeSpan {
    return TimeSpan.Minutes(value.valueOf() * 60);
  }

  /** Represents an hour. */
  static readonly Hour = TimeSpan.Hours(1);

  /**
   * Creates a TimeSpan representing the given number of days.
   *
   * @param value - The number of days.
   * @returns A TimeSpan representing the given number of days.
   */
  static Days(value: UnparsedTimeStamp = 1): TimeSpan {
    return TimeSpan.Hours(value.valueOf() * 24);
  }

  /** Represents a day. */
  static readonly Day = TimeSpan.Days(1);

  /** The maximum possible value for a TimeSpan. */
  static readonly Max = new TimeSpan(this.MAX_SAFE_INTEGER);

  /** The minimum possible value for a TimeSpan. */
  static readonly Min = new TimeSpan(-this.MAX_SAFE_INTEGER);

  /** The zero value for a TimeSpan. */
  static readonly Zero = new TimeSpan(0);
}

/** Rate represents a data rate in Hz. */
export class Rate extends Number {
  constructor(value: UnparsedRate) {
    if (value instanceof Number) super(value.valueOf());
    else super(value);
  }

  /** @returns The number of seconds in the Rate. */
  equals(other: UnparsedRate): boolean {
    return this.valueOf() === new Rate(other).valueOf();
  }

  /**
   * Calculates the period of the Rate as a TimeSpan.
   *
   * @returns A TimeSpan representing the period of the Rate.
   */
  period(): TimeSpan {
    return new TimeSpan(TimeSpan.Seconds(this.valueOf()).valueOf());
  }

  /**
   * Calculates the number of samples in the given TimeSpan at this rate.
   *
   * @param duration - The duration to calculate the sample count from.
   * @returns The number of samples in the given TimeSpan at this rate.
   */
  sampleCount(duration: UnparsedTimeSpan): number {
    return new TimeSpan(duration).seconds() * this.valueOf();
  }

  /**
   * Calculates the number of bytes in the given TimeSpan at this rate.
   *
   * @param span - The duration to calculate the byte count from.
   * @param density - The density of the data in bytes per sample.
   * @returns The number of bytes in the given TimeSpan at this rate.
   */
  byteCount(span: UnparsedTimeSpan, density: UnparsedDensity): number {
    return this.sampleCount(span) * new Density(density).valueOf();
  }

  /**
   * Calculates a TimeSpan given the number of samples at this rate.
   *
   * @param sampleCount - The number of samples in the span.
   * @returns A TimeSpan that corresponds to the given number of samples.
   */
  span(sampleCount: number): TimeSpan {
    return TimeSpan.Seconds(sampleCount / this.valueOf());
  }

  /**
   * Calculates a TimeSpan given the number of bytes at this rate.
   *
   * @param size - The number of bytes in the span.
   * @param density - The density of the data in bytes per sample.
   * @returns A TimeSpan that corresponds to the given number of bytes.
   */
  byteSpan(size: Size, density: UnparsedDensity): TimeSpan {
    return this.span(size.valueOf() / density.valueOf());
  }

  /**
   * Creates a Rate representing the given number of Hz.
   *
   * @param value - The number of Hz.
   * @returns A Rate representing the given number of Hz.
   */
  static Hz(value: number): Rate {
    return new Rate(value);
  }

  /**
   * Creates a Rate representing the given number of kHz.
   *
   * @param value - The number of kHz.
   * @returns A Rate representing the given number of kHz.
   */
  static KHz(value: number): Rate {
    return Rate.Hz(value * 1000);
  }
}

/** Density represents the number of bytes in a value. */
export class Density extends Number {
  /**
   * Creates a Density representing the given number of bytes per value.
   *
   * @class
   * @param value - The number of bytes per value.
   * @returns A Density representing the given number of bytes per value.
   */
  constructor(value: UnparsedDensity) {
    if (value instanceof Number) super(value.valueOf());
    else super(value);
  }

  /** Represents an Unknown/Invalid Density. */
  static readonly Unknown = new Density(0);
  /** Represents a Density of 64 bits per value. */
  static readonly Bit64 = new Density(8);
  /** Represents a Density of 32 bits per value. */
  static readonly Bit32 = new Density(4);
  /** Represents a Density of 16 bits per value. */
  static readonly Bit16 = new Density(2);
  /** Represents a Density of 8 bits per value. */
  static readonly Bit8 = new Density(1);
}

/**
 * TimeRange represents a range of time between two TimeStamps. It's important
 * to note that the start of the range is inclusive, while the end of the range
 * is exclusive.
 *
 * @property start - A TimeStamp representing the start of the range.
 * @property end - A Timestamp representing the end of the range.
 */
export class TimeRange {
  start: TimeStamp;
  end: TimeStamp;

  /**
   * Creates a TimeRange from the given start and end TimeStamps.
   *
   * @param start - A TimeStamp representing the start of the range.
   * @param end - A TimeStamp representing the end of the range.
   */
  constructor(start: UnparsedTimeStamp, end: UnparsedTimeStamp) {
    this.start = new TimeStamp(start);
    this.end = new TimeStamp(end);
  }

  /** @returns The TimeSpan occupied by the TimeRange. */
  span(): TimeSpan {
    return new TimeSpan(this.end.valueOf() - this.start.valueOf());
  }

  /**
   * Checks if the timestamp is valid i.e. the start is before the end.
   *
   * @returns True if the TimeRange is valid.
   */
  isValid(): boolean {
    return this.start.valueOf() <= this.end.valueOf();
  }

  /**
   * Makes sure the TimeRange is valid i.e. the start is before the end.
   *
   * @returns A TimeRange that is valid.
   */
  makeValid(): TimeRange {
    return this.isValid() ? this : this.swap();
  }

  /**
   * Checks if the TimeRange has a zero span.
   *
   * @returns True if the TimeRange has a zero span.
   */
  isZero(): boolean {
    return this.span().isZero();
  }

  /**
   * Creates a new TimeRange with the start and end swapped.
   *
   * @returns A TimeRange with the start and end swapped.
   */
  swap(): TimeRange {
    return new TimeRange(this.end, this.start);
  }

  /**
   * Checks if the TimeRange is equal to the given TimeRange.
   *
   * @param other - The TimeRange to compare to.
   * @returns True if the TimeRange is equal to the given TimeRange.
   */
  equals(other: TimeRange): boolean {
    return this.start.equals(other.start) && this.end.equals(other.end);
  }

  static readonly Max = new TimeRange(TimeStamp.MIN, TimeStamp.MAX);
}

/** DataType is a string that represents a data type. */
export class DataType extends String {
  constructor(value: UnparsedDataType) {
    if (typeof value === "string") {
      super(value);
    } else {
      super(value.valueOf());
    }
  }

  get Array(): TypedArrayConstructor {
    const v = ARRAY_CONSTRUCTORS.get(this.string);
    if (v === undefined) {
      throw new Error(`Unknown data type: ${this.string}`);
    }
    return v;
  }

  get string(): string {
    return this.valueOf();
  }

  get density(): Density {
    const v = DATA_TYPE_DENSITIES.get(this.string);
    if (v === undefined) {
      throw new Error(`Unknown data type: ${this.string}`);
    }
    return v;
  }

  checkArray(array: TypedArray): boolean {
    return array.constructor === this.Array;
  }

  toJSON(): string {
    return this.string;
  }

  /** Represents an Unknown/Invalid DataType. */
  static readonly Unknown = new DataType("unknown");
  /** Represents a 64-bit floating point value. */
  static readonly Float64 = new DataType("float64");
  /** Represents a 32-bit floating point value. */
  static readonly Float32 = new DataType("float32");
  /** Represents a 64-bit signed integer value. */
  static readonly Int64 = new DataType("int64");
  /** Represents a 32-bit signed integer value. */
  static readonly Int32 = new DataType("int32");
  /** Represents a 16-bit signed integer value. */
  static readonly Int16 = new DataType("int16");
  /** Represents a 8-bit signed integer value. */
  static readonly Int8 = new DataType("int8");
  /** Represents a 64-bit unsigned integer value. */
  static readonly Uint64 = new DataType("uint64");
  /** Represents a 32-bit unsigned integer value. */
  static readonly Uint32 = new DataType("uint32");
  /** Represents a 16-bit unsigned integer value. */
  static readonly Uint16 = new DataType("uint16");
  /** Represents a 8-bit unsigned integer value. */
  static readonly Uint8 = new DataType("uint8");
  /** Represents a 64-bit unix epoch. */
  static readonly TimeStamp = new DataType("timestamp");
}

export class Size extends Number {
  constructor(value: UnparsedSize) {
    super(value.valueOf());
  }

  largerThan(other: Size): boolean {
    return this.valueOf() > other.valueOf();
  }

  smallerThan(other: Size): boolean {
    return this.valueOf() < other.valueOf();
  }

  static Bytes(value: UnparsedSize): Size {
    return new Size(value);
  }

  static readonly Byte = new Size(1);

  static Kilobytes(value: UnparsedSize): Size {
    return Size.Bytes(value.valueOf() * 1e3);
  }

  static readonly Kilobyte = Size.Kilobytes(1);

  static Megabytes(value: UnparsedSize): Size {
    return Size.Kilobytes(value.valueOf() * 1e3);
  }

  static readonly Megabyte = Size.Megabytes(1);

  static Gigabytes(value: UnparsedSize): Size {
    return Size.Megabytes(value.valueOf() * 1e3);
  }

  static readonly Gigabyte = Size.Gigabytes(1);

  static Terabytes(value: UnparsedSize): Size {
    return Size.Gigabytes(value.valueOf() * 1e3);
  }

  static readonly Terabyte = Size.Terabytes(1);
}

export type UnparsedTimeStamp = TimeStamp | TimeSpan | number;
export type UnparsedTimeSpan = TimeSpan | TimeStamp | number;
export type UnparsedRate = Rate | number;
export type UnparsedDensity = Density | number;
export type UnparsedDataType = DataType | string;
export type UnparsedSize = Size | number;

registerCustomTypeEncoder({ Class: TimeStamp, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: TimeSpan, write: valueOfEncoder });
registerCustomTypeEncoder({
  Class: DataType,
  write: (v: unknown) => (v as DataType).string,
});
registerCustomTypeEncoder({ Class: Rate, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: Density, write: valueOfEncoder });

export type TypedArray =
  | Uint8Array
  | Uint16Array
  | Uint32Array
  | BigUint64Array
  | Float32Array
  | Float64Array
  | Int8Array
  | Int16Array
  | Int32Array
  | BigInt64Array;

type TypedArrayConstructor =
  | Uint8ArrayConstructor
  | Uint16ArrayConstructor
  | Uint32ArrayConstructor
  | BigUint64ArrayConstructor
  | Float32ArrayConstructor
  | Float64ArrayConstructor
  | Int8ArrayConstructor
  | Int16ArrayConstructor
  | Int32ArrayConstructor
  | BigInt64ArrayConstructor;

const ARRAY_CONSTRUCTORS: Map<string, TypedArrayConstructor> = new Map<
  string,
  TypedArrayConstructor
>([
  [DataType.Uint8.string, Uint8Array],
  [DataType.Uint16.string, Uint16Array],
  [DataType.Uint32.string, Uint32Array],
  [DataType.Uint64.string, BigUint64Array],
  [DataType.Float32.string, Float32Array],
  [DataType.Float64.string, Float64Array],
  [DataType.Int8.string, Int8Array],
  [DataType.Int16.string, Int16Array],
  [DataType.Int32.string, Int32Array],
  [DataType.Int64.string, BigInt64Array],
  [DataType.TimeStamp.string, BigInt64Array],
]);

const DATA_TYPE_DENSITIES = new Map<string, Density>([
  [DataType.Uint8.string, Density.Bit8],
  [DataType.Uint16.string, Density.Bit16],
  [DataType.Uint32.string, Density.Bit32],
  [DataType.Uint64.string, Density.Bit64],
  [DataType.Float32.string, Density.Bit32],
  [DataType.Float64.string, Density.Bit64],
  [DataType.Int8.string, Density.Bit8],
  [DataType.Int16.string, Density.Bit16],
  [DataType.Int32.string, Density.Bit32],
  [DataType.Int64.string, Density.Bit64],
  [DataType.TimeStamp.string, Density.Bit64],
]);
