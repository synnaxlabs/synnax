// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { math } from "@/math";
import { primitive } from "@/primitive";
import { type bounds } from "@/spatial";

/** Time zone specification when working with time stamps. */
export type TZInfo = "UTC" | "local";

const SIMPLE_DAYS_IN_YEAR = 365;
const SIMPLE_DAYS_IN_MONTH = 30;

/** Different string formats for time stamps. */
export type TimeStampStringFormat =
  | "ISO"
  | "ISODate"
  | "ISOTime"
  | "time"
  | "preciseTime"
  | "date"
  | "preciseDate"
  | "shortDate"
  | "dateTime";

/** Different string formats for time spans. */
export type TimeSpanStringFormat = "full" | "semantic";

const dateComponentsZ = z.union([
  z.tuple([z.int()]),
  z.tuple([z.int(), z.int().min(1).max(12)]),
  z.tuple([z.int(), z.int().min(1).max(12), z.int().min(1).max(31)]),
]);

/**
 * A triple of numbers representing a date.
 *
 * @param year - The year.
 * @param month - The month.
 * @param day - The day.
 */
export type DateComponents = z.infer<typeof dateComponentsZ>;

const remainder = <T extends TimeStamp | TimeSpan>(
  value: T,
  divisor: TimeSpan | TimeStamp,
): T => {
  const ts = new TimeStamp(divisor);
  if (
    ![
      TimeSpan.DAY,
      TimeSpan.HOUR,
      TimeSpan.MINUTE,
      TimeSpan.SECOND,
      TimeSpan.MILLISECOND,
      TimeSpan.MICROSECOND,
      TimeSpan.NANOSECOND,
    ].some((s) => s.equals(ts))
  )
    throw new Error(
      "Invalid argument for remainder. Must be an even TimeSpan or Timestamp",
    );
  const v = value.valueOf() % ts.valueOf();
  return (value instanceof TimeStamp ? new TimeStamp(v) : new TimeSpan(v)) as T;
};

/**
UTC timestamp. Synnax uses a nanosecond precision int64 timestamp.
 *
 * @param value - The timestamp value to parse. This can be any of the following:
 *
 * 1. A number representing the number of nanoseconds since the Unix epoch.
 * 2. A JavaScript Date object.
 * 3. An array of numbers satisfying the DateComponents type, where the first element is the
 *   year, the second is the month, and the third is the day. To increase resolution
 *   when using this method, use the add method. It's important to note that this initializes
 *   a timestamp at midnight UTC, regardless of the timezone specified.
 * 4. An ISO compliant date or date time string. The time zone component is ignored.
 *
 * @param tzInfo - The timezone to use when parsing the timestamp. This can be either "UTC" or
 * "local". This parameter is ignored if the value is a Date object or a DateComponents array.
 *
 * @example ts = new TimeStamp(1 * TimeSpan.HOUR) // 1 hour after the Unix epoch
 * @example ts = new TimeStamp([2021, 1, 1]) // 1/1/2021 at midnight UTC
 * @example ts = new TimeStamp([2021, 1, 1]).add(1 * TimeSpan.HOUR) // 1/1/2021 at 1am UTC
 * @example ts = new TimeStamp("2021-01-01T12:30:00Z") // 1/1/2021 at 12:30pm UTC
 */
export class TimeStamp
  extends primitive.ValueExtension<bigint>
  implements primitive.Stringer
{
  constructor(value?: CrudeTimeStamp, tzInfo: TZInfo = "UTC") {
    if (value == null) super(TimeStamp.now().valueOf());
    else if (value instanceof Date)
      super(BigInt(value.getTime()) * TimeStamp.MILLISECOND.valueOf());
    else if (typeof value === "string")
      super(TimeStamp.parseDateTimeString(value, tzInfo).valueOf());
    else if (Array.isArray(value)) super(TimeStamp.parseDate(value));
    else {
      let offset = 0n;
      if (value instanceof Number) value = value.valueOf();
      if (tzInfo === "local") offset = TimeStamp.utcOffset.valueOf();
      if (typeof value === "number")
        if (isFinite(value))
          if (value === math.MAX_INT64_NUMBER) value = math.MAX_INT64;
          else value = Math.trunc(value);
        else {
          if (isNaN(value)) value = 0;
          if (value === Infinity) value = TimeStamp.MAX;
          else value = TimeStamp.MIN;
        }
      if (primitive.isCrudeValueExtension<bigint>(value)) value = value.value;
      super(BigInt(value.valueOf()) + offset);
    }
  }

  private static parseDate([year = 1970, month = 1, day = 1]: DateComponents): bigint {
    const date = new Date(year, month - 1, day, 0, 0, 0, 0);
    // We truncate here to only get the date component regardless of the time zone.
    return new TimeStamp(BigInt(date.getTime()) * TimeStamp.MILLISECOND.valueOf())
      .truncate(TimeStamp.DAY)
      .valueOf();
  }

  /**
   * @returns the primitive value of the TimeStamp. Overrides standard JS valueOf()
   * method.
   */
  valueOf(): bigint {
    return this.value;
  }

  private static parseTimeString(time: string, tzInfo: TZInfo = "UTC"): bigint {
    const [hours, minutes, mbeSeconds] = time.split(":");
    let seconds = "00";
    let milliseconds: string | undefined = "00";
    if (mbeSeconds != null) [seconds, milliseconds] = mbeSeconds.split(".");
    let base = TimeStamp.hours(parseInt(hours ?? "00"))
      .add(TimeStamp.minutes(parseInt(minutes ?? "00")))
      .add(TimeStamp.seconds(parseInt(seconds ?? "00")))
      .add(TimeStamp.milliseconds(parseInt(milliseconds ?? "00")));
    if (tzInfo === "local") base = base.add(TimeStamp.utcOffset);
    return base.valueOf();
  }

  private static parseDateTimeString(str: string, tzInfo: TZInfo = "UTC"): bigint {
    if (!str.includes("/") && !str.includes("-"))
      return TimeStamp.parseTimeString(str, tzInfo);

    const isDateTimeLocal =
      str.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,3})?$/) != null;

    if (isDateTimeLocal) {
      let datePart = str;
      let ms = 0;

      if (str.includes(".")) {
        const parts = str.split(".");
        datePart = parts[0];
        const msPart = parts[1] || "0";
        ms = parseInt(msPart.padEnd(3, "0").slice(0, 3));
      }

      const d =
        tzInfo === "local"
          ? new Date(datePart.replace("T", " "))
          : new Date(`${datePart}Z`);

      const baseBigInt = BigInt(d.getTime()) * TimeStamp.MILLISECOND.valueOf();
      const msBigInt = BigInt(ms) * TimeStamp.MILLISECOND.valueOf();

      return baseBigInt + msBigInt;
    }

    const d = new Date(str);
    // Essential to note that this makes the date midnight in UTC! Not local!
    // As a result, we need to add the tzInfo offset back in.
    if (!str.includes(":")) d.setUTCHours(0, 0, 0, 0);
    return new TimeStamp(
      BigInt(d.getTime()) * TimeStamp.MILLISECOND.valueOf(),
      tzInfo,
    ).valueOf();
  }

  private toISOString(tzInfo: TZInfo = "UTC"): string {
    if (tzInfo === "UTC") return this.date().toISOString();
    return this.sub(TimeStamp.utcOffset).date().toISOString();
  }

  private timeString(milliseconds: boolean = false, tzInfo: TZInfo = "UTC"): string {
    const iso = this.toISOString(tzInfo);
    return milliseconds ? iso.slice(11, 23) : iso.slice(11, 19);
  }

  private dateString(): string {
    const date = this.date();
    const month = date.toLocaleString("default", { month: "short" });
    const day = date.toLocaleString("default", { day: "numeric" });
    return `${month} ${day}`;
  }

  /** @returns The UTC offset for the time zone of the machine. */
  static get utcOffset(): TimeSpan {
    return new TimeSpan(
      BigInt(new Date().getTimezoneOffset()) * TimeStamp.MINUTE.valueOf(),
    );
  }

  /**
   * @returns a TimeSpan representing the amount time elapsed since
   * the other timestamp.
   * @param other - The other timestamp.
   */
  static since(other: CrudeTimeStamp): TimeSpan {
    return new TimeStamp().span(other);
  }

  /** @returns A JavaScript Date object representing the TimeStamp. */
  date(): Date {
    return new Date(this.milliseconds);
  }

  /**
   * Checks if the TimeStamp is equal to another TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamps are equal, false otherwise.
   */
  equals(other: CrudeTimeStamp): boolean {
    return this.valueOf() === new TimeStamp(other).valueOf();
  }

  /**
   * Creates a TimeSpan representing the duration between the two timestamps.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns A TimeSpan representing the duration between the two timestamps.
   *   The span is guaranteed to be positive.
   */
  span(other: CrudeTimeStamp): TimeSpan {
    return this.range(other).span;
  }

  /**
   * Creates a TimeRange spanning the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns A TimeRange spanning the given TimeStamp that is guaranteed to be
   *   valid, regardless of the TimeStamp order.
   */
  range(other: CrudeTimeStamp): TimeRange {
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
  spanRange(other: CrudeTimeSpan): TimeRange {
    return this.range(this.add(other)).makeValid();
  }

  /**
   * Checks if the TimeStamp represents the unix epoch.
   *
   * @returns True if the TimeStamp represents the unix epoch, false otherwise.
   */
  get isZero(): boolean {
    return this.valueOf() === 0n;
  }

  /**
   * Checks if the TimeStamp is after the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamp is after the given TimeStamp, false
   *   otherwise.
   */
  after(other: CrudeTimeStamp): boolean {
    return this.valueOf() > new TimeStamp(other).valueOf();
  }

  /**
   * Checks if the TimeStamp is after or equal to the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamp is after or equal to the given TimeStamp,
   *   false otherwise.
   */
  afterEq(other: CrudeTimeStamp): boolean {
    return this.valueOf() >= new TimeStamp(other).valueOf();
  }

  /**
   * Checks if the TimeStamp is before the given TimeStamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if the TimeStamp is before the given TimeStamp, false
   *   otherwise.
   */
  before(other: CrudeTimeStamp): boolean {
    return this.valueOf() < new TimeStamp(other).valueOf();
  }

  /**
   * Checks if TimeStamp is before or equal to the current timestamp.
   *
   * @param other - The other TimeStamp to compare to.
   * @returns True if TimeStamp is before or equal to the current timestamp,
   *   false otherwise.
   */
  beforeEq(other: CrudeTimeStamp): boolean {
    return this.valueOf() <= new TimeStamp(other).valueOf();
  }

  /**
   * Adds a TimeSpan to the TimeStamp.
   *
   * @param span - The TimeSpan to add.
   * @returns A new TimeStamp representing the sum of the TimeStamp and
   *   TimeSpan.
   */
  add(span: CrudeTimeSpan): TimeStamp {
    return new TimeStamp(math.add(this.valueOf(), new TimeSpan(span).valueOf()));
  }

  /**
   * Subtracts a TimeSpan from the TimeStamp.
   *
   * @param span - The TimeSpan to subtract.
   * @returns A new TimeStamp representing the difference of the TimeStamp and
   *   TimeSpan.
   */
  sub(span: CrudeTimeSpan): TimeStamp {
    return new TimeStamp(math.sub(this.valueOf(), new TimeSpan(span).valueOf()));
  }

  /**
   * @returns the floating point number of hours since the unix epoch to the timestamp
   * value.
   */
  get hours(): number {
    return Number(this.valueOf()) / Number(TimeSpan.HOUR.valueOf());
  }

  /**
   * @returns the floating point number of minutes since the unix epoch to the timestamp
   * value.
   */
  get minutes(): number {
    return Number(this.valueOf()) / Number(TimeSpan.MINUTE.valueOf());
  }

  /**
   * @returns the floating point number of days since the unix epoch to the timestamp
   * value.
   */
  get days(): number {
    return Number(this.valueOf()) / Number(TimeSpan.DAY.valueOf());
  }

  /**
   * @returns the floating point number of seconds since the unix epoch to the timestamp
   * value.
   */
  get seconds(): number {
    return Number(this.valueOf()) / Number(TimeSpan.SECOND.valueOf());
  }

  /** @returns the floating point number of milliseconds since the unix epoch. */
  get milliseconds(): number {
    return Number(this.valueOf()) / Number(TimeStamp.MILLISECOND.valueOf());
  }

  /** @returns the floating point number of microseconds since the unix epoch. */
  get microseconds(): number {
    return Number(this.valueOf()) / Number(TimeStamp.MICROSECOND.valueOf());
  }

  /**
   * @returns the floating point number of nanoseconds since the unix epoch.
   * Note that since we're converting to float64, this reduces the resolution
   * to a quarter of a microsecond.
   */
  get nanoseconds(): number {
    return Number(this.valueOf());
  }

  /** @returns the integer year that the timestamp corresponds to. */
  get year(): number {
    return this.date().getUTCFullYear();
  }

  /**
   * @returns a copy of the timestamp with the year changed.
   * @param year the value to set the year to.
   */
  setYear(year: number): TimeStamp {
    const d = this.date();
    d.setUTCFullYear(year);
    return new TimeStamp(d);
  }

  /** @returns the integer month that the timestamp corresponds to with its year. */
  get month(): number {
    return this.date().getUTCMonth();
  }

  /**
   * @returns a copy of the timestamp with the month changed.
   * @param month the value to set the month to.
   */
  setMonth(month: number): TimeStamp {
    const d = this.date();
    d.setUTCMonth(month);
    return new TimeStamp(d);
  }

  /** @returns the integer day that the timestamp corresponds to within its month. */
  get day(): number {
    return this.date().getUTCDate();
  }

  /**
   * @returns a copy of the timestamp with the day changed.
   * @param day the value the set the day to.
   */
  setDay(day: number): TimeStamp {
    const d = this.date();
    d.setUTCDate(day);
    return new TimeStamp(d);
  }

  /**
   * @returns the integer hour that the timestamp corresponds to within its day.
   */
  get hour(): number {
    return this.date().getUTCHours();
  }

  /**
   * @returns the integer hour that the timestamp corresponds to within its day in local time.
   */
  get localHour(): number {
    return this.date().getHours();
  }

  /**
   * @returns a copy of the timestamp with the hour changed.
   * @param hour the value to set the hour to.
   */
  setLocalHour(hour: number): TimeStamp {
    const d = this.date();
    d.setHours(hour);
    return new TimeStamp(d);
  }

  /**
   * @returns a copy of the timestamp with the hour changed.
   * @param hour the value to set the hour to.
   */
  setHour(hour: number): TimeStamp {
    const d = this.date();
    d.setUTCHours(hour);
    return new TimeStamp(d);
  }

  /** @returns the integer minute that the timestamp corresponds to within its hour. */
  get minute(): number {
    return this.date().getUTCMinutes();
  }

  /**
   * @returns a copy of the timestamp with the minute changed.
   * @param minute the value to set the minute to.
   */
  setMinute(minute: number): TimeStamp {
    const d = this.date();
    d.setUTCMinutes(minute);
    return new TimeStamp(d);
  }

  /**
   * @returns the integer second that the timestamp corresponds to within its
   * minute.
   */
  get second(): number {
    return this.date().getUTCSeconds();
  }

  /**
   * @returns a copy of the timestamp with the second changed.
   * @param second the value to set the second to.
   */
  setSecond(second: number): TimeStamp {
    const d = this.date();
    d.setUTCSeconds(second);
    return new TimeStamp(d);
  }

  /**
   * @returns the integer millisecond that the timestamp corresponds to within its
   * second.
   */
  get millisecond(): number {
    return this.date().getUTCMilliseconds();
  }

  /**
   * @returns a copy of the timestamp with the milliseconds changed.
   * @param millisecond the value to set the millisecond to.
   */
  setMillisecond(millisecond: number): TimeStamp {
    const d = this.date();
    d.setUTCMilliseconds(millisecond);
    return new TimeStamp(d);
  }

  /**
   * Returns a string representation of the TimeStamp.
   *
   * @param format - Optional format for the string representation. Defaults to "ISO".
   * @param tzInfo - Optional timezone info. Defaults to "UTC".
   * @returns A string representation of the TimeStamp.
   */
  toString(format: TimeStampStringFormat = "ISO", tzInfo: TZInfo = "UTC"): string {
    switch (format) {
      case "ISODate":
        return this.toISOString(tzInfo).slice(0, 10);
      case "ISOTime":
        return this.toISOString(tzInfo).slice(11, 23);
      case "time":
        return this.timeString(false, tzInfo);
      case "preciseTime":
        return this.timeString(true, tzInfo);
      case "date":
        return this.dateString();
      case "preciseDate":
        return `${this.dateString()} ${this.timeString(true, tzInfo)}`;
      case "dateTime":
        return `${this.dateString()} ${this.timeString(false, tzInfo)}`;
      default:
        return this.toISOString(tzInfo);
    }
  }

  /**
   * @returns A new TimeStamp that is the remainder of the TimeStamp divided by the
   * given span. This is useful in cases where you want only part of a TimeStamp's value
   * i.e., the hours, minutes, seconds, milliseconds, microseconds, and nanoseconds but
   * not the days, years, etc.
   *
   * @param divisor - The TimeSpan to divide by. Must be an even TimeSpan or TimeStamp. Even
   * means it must be a day, hour, minute, second, millisecond, or microsecond, etc.
   *
   * @example TimeStamp.now().remainder(TimeStamp.DAY) // => TimeStamp representing the current day
   */
  remainder(divisor: TimeSpan | TimeStamp): TimeStamp {
    return remainder(this, divisor);
  }

  /** @returns true if the day portion TimeStamp is today, false otherwise. */
  get isToday(): boolean {
    return this.truncate(TimeSpan.DAY).equals(TimeStamp.now().truncate(TimeSpan.DAY));
  }

  /**
   * Truncates the TimeStamp to the nearest multiple of the given span.
   *
   * @param span - The TimeSpan to truncate to.
   * @returns A new TimeStamp that is truncated to the nearest multiple of the given span.
   */
  truncate(span: TimeSpan | TimeStamp): TimeStamp {
    return this.sub(this.remainder(span));
  }

  /**
   * Determines the appropriate string format based on the span magnitude.
   *
   * @param span - The span that provides context for format selection
   * @returns The appropriate TimeStampStringFormat
   *
   * Rules:
   * - For spans >= 30 days: "shortDate" (e.g., "Nov 5")
   * - For spans >= 1 day: "dateTime" (e.g., "Nov 5 14:23:45")
   * - For spans >= 1 hour: "time" (e.g., "14:23:45")
   * - For spans >= 1 second: "preciseTime" (e.g., "14:23:45.123")
   * - For spans < 1 second: "ISOTime" (full precision time)
   */
  formatBySpan(span: TimeSpan): TimeStampStringFormat {
    if (span.greaterThanOrEqual(TimeSpan.days(30))) return "shortDate";
    if (span.greaterThanOrEqual(TimeSpan.DAY)) return "dateTime";
    if (span.greaterThanOrEqual(TimeSpan.HOUR)) return "time";
    if (span.greaterThanOrEqual(TimeSpan.SECOND)) return "preciseTime";

    return "ISOTime";
  }

  /**
   * @returns A new TimeStamp representing the current time in UTC. It's important to
   * note that this TimeStamp is only accurate to the millisecond level (that's the best
   * JavaScript can do).
   */
  static now(): TimeStamp {
    return new TimeStamp(new Date());
  }

  /**
   * Finds the maximum timestamp among the provided timestamps.
   *
   * @param timestamps - The timestamps to compare.
   * @returns The maximum (latest) timestamp from the input.
   */
  static max(...timestamps: CrudeTimeStamp[]): TimeStamp {
    let max = TimeStamp.MIN;
    for (const ts of timestamps) {
      const t = new TimeStamp(ts);
      if (t.after(max)) max = t;
    }
    return max;
  }

  /**
   * Finds the minimum timestamp among the provided timestamps.
   *
   * @param timestamps - The timestamps to compare.
   * @returns The minimum (earliest) timestamp from the input.
   */
  static min(...timestamps: CrudeTimeStamp[]): TimeStamp {
    let min = TimeStamp.MAX;
    for (const ts of timestamps) {
      const t = new TimeStamp(ts);
      if (t.before(min)) min = t;
    }
    return min;
  }

  /**
   * Creates a TimeStamp representing the given number of nanoseconds.
   *
   * @param value - The number of nanoseconds.
   * @returns A TimeStamp representing the given number of nanoseconds.
   */
  static nanoseconds(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return new TimeStamp(value, tzInfo);
  }

  /** One nanosecond after the unix epoch */
  static readonly NANOSECOND = TimeStamp.nanoseconds(1);

  /** @returns a new TimeStamp n microseconds after the unix epoch */
  static microseconds(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return TimeStamp.nanoseconds(value * 1000, tzInfo);
  }

  /** One microsecond after the unix epoch */
  static readonly MICROSECOND = TimeStamp.microseconds(1);

  /** @returns a new TimeStamp n milliseconds after the unix epoch */
  static milliseconds(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return TimeStamp.microseconds(value * 1000, tzInfo);
  }

  /** One millisecond after the unix epoch */
  static readonly MILLISECOND = TimeStamp.milliseconds(1);

  /** @returns a new TimeStamp n seconds after the unix epoch */
  static seconds(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return TimeStamp.milliseconds(value * 1000, tzInfo);
  }

  /** One second after the unix epoch */
  static readonly SECOND = TimeStamp.seconds(1);

  /** @returns a new TimeStamp n minutes after the unix epoch */
  static minutes(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return TimeStamp.seconds(value * 60, tzInfo);
  }

  /** One minute after the unix epoch */
  static readonly MINUTE = TimeStamp.minutes(1);

  /** @returns a new TimeStamp n hours after the unix epoch */
  static hours(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return TimeStamp.minutes(value * 60, tzInfo);
  }

  /** One hour after the unix epoch */
  static readonly HOUR = TimeStamp.hours(1);

  /** @returns a new TimeStamp n days after the unix epoch */
  static days(value: number, tzInfo: TZInfo = "UTC"): TimeStamp {
    return TimeStamp.hours(value * 24, tzInfo);
  }

  /** One day after the unix epoch */
  static readonly DAY = TimeStamp.days(1);

  /** The maximum possible value for a timestamp */
  static readonly MAX = new TimeStamp(math.MAX_INT64);

  /** The minimum possible value for a timestamp */
  static readonly MIN = new TimeStamp(0);

  /** The unix epoch */
  static readonly ZERO = new TimeStamp(0);

  /** A zod schema for validating timestamps */
  static readonly z = z.union([
    z.instanceof(TimeStamp),
    z.object({ value: z.bigint() }).transform((v) => new TimeStamp(v.value)),
    z.string().transform((n) => new TimeStamp(BigInt(n))),
    z.number().transform((n) => new TimeStamp(n)),
    z.bigint().transform((n) => new TimeStamp(n)),
    z.date().transform((d) => new TimeStamp(d)),
    z.custom<TimeSpan>((v) => v instanceof TimeSpan).transform((v) => new TimeStamp(v)),
    dateComponentsZ.transform((v) => new TimeStamp(v)),
  ]);

  /**
   * Sorts two timestamps.
   *
   * @param a - The first timestamp.
   * @param b - The second timestamp.
   * @returns A number indicating the order of the two timestamps (positive if a is
   * greater than b, negative if a is less than b, and 0 if they are equal).
   */
  static sort(a: TimeStamp, b: TimeStamp): number {
    return Number(a.valueOf() - b.valueOf());
  }
}

/** TimeSpan represents a nanosecond precision duration. */
export class TimeSpan
  extends primitive.ValueExtension<bigint>
  implements primitive.Stringer
{
  constructor(value: CrudeTimeSpan) {
    if (typeof value === "number") value = Math.trunc(value.valueOf());
    if (primitive.isCrudeValueExtension<bigint>(value)) value = value.value;
    super(BigInt(value.valueOf()));
  }

  /**
   * Creates a TimeSpan representing the given number of seconds.
   *
   * @param span - The number of seconds.
   * @returns A TimeSpan representing the given number of seconds.
   */
  static fromSeconds(span: CrudeTimeSpan): TimeSpan {
    if (span instanceof TimeSpan) return span;
    if (span instanceof Rate) return span.period;
    if (span instanceof TimeStamp) return new TimeSpan(span);
    if (primitive.isCrudeValueExtension<bigint>(span)) span = span.value;
    if (["number", "bigint"].includes(typeof span)) return TimeSpan.seconds(span);
    return new TimeSpan(span);
  }

  /**
   * Creates a TimeSpan representing the given number of milliseconds.
   *
   * @param span - The number of milliseconds.
   * @returns A TimeSpan representing the given number of milliseconds.
   */
  static fromMilliseconds(span: CrudeTimeSpan): TimeSpan {
    if (span instanceof TimeSpan) return span;
    if (span instanceof Rate) return span.period;
    if (span instanceof TimeStamp) return new TimeSpan(span);
    if (primitive.isCrudeValueExtension<bigint>(span)) span = span.value;
    if (["number", "bigint"].includes(typeof span)) return TimeSpan.milliseconds(span);
    return new TimeSpan(span);
  }

  /**
   * @returns the primitive value of the TimeSpan. Overrides standard JS valueOf()
   * method.
   */
  valueOf(): bigint {
    return this.value;
  }

  /**
   * Checks if the TimeSpan is less than another TimeSpan.
   *
   * @param other - The TimeSpan to compare against.
   * @returns True if the TimeSpan is less than the other TimeSpan, false otherwise.
   */
  lessThan(other: CrudeTimeSpan): boolean {
    return this.valueOf() < new TimeSpan(other).valueOf();
  }

  /**
   * Checks if the TimeSpan is greater than another TimeSpan.
   *
   * @param other - The TimeSpan to compare against.
   * @returns True if the TimeSpan is greater than the other TimeSpan, false otherwise.
   */
  greaterThan(other: CrudeTimeSpan): boolean {
    return this.valueOf() > new TimeSpan(other).valueOf();
  }

  /**
   * Checks if the TimeSpan is less than or equal to another TimeSpan.
   *
   * @param other - The TimeSpan to compare against.
   * @returns True if the TimeSpan is less than or equal to the other TimeSpan, false otherwise.
   */
  lessThanOrEqual(other: CrudeTimeSpan): boolean {
    return this.valueOf() <= new TimeSpan(other).valueOf();
  }

  /**
   * Checks if the TimeSpan is greater than or equal to another TimeSpan.
   *
   * @param other - The TimeSpan to compare against.
   * @returns True if the TimeSpan is greater than or equal to the other TimeSpan, false otherwise.
   */
  greaterThanOrEqual(other: CrudeTimeSpan): boolean {
    return this.valueOf() >= new TimeSpan(other).valueOf();
  }

  /**
   * Calculates the remainder of the TimeSpan when divided by another TimeSpan.
   *
   * @param divisor - The TimeSpan to divide by.
   * @returns A new TimeSpan representing the remainder.
   */
  remainder(divisor: TimeSpan): TimeSpan {
    return remainder(this, divisor);
  }

  /**
   * Truncates the TimeSpan to the nearest multiple of the given span.
   *
   * @param span - The TimeSpan to truncate to.
   * @returns A new TimeSpan that is truncated to the nearest multiple of the given span.
   */
  truncate(span: TimeSpan): TimeSpan {
    return new TimeSpan(
      BigInt(Math.trunc(Number(this.valueOf() / span.valueOf()))) * span.valueOf(),
    );
  }

  /**
   * Returns a string representation of the TimeSpan.
   *
   * @param format - Optional format for the string representation. Defaults to "full".
   *   - "full": Shows all non-zero units with full precision (e.g., "2d 3h 45m 12s 500ms")
   *   - "semantic": Shows 1-2 most significant units (e.g., "2d 3h")
   * @returns A string representation of the TimeSpan.
   */
  toString(format: TimeSpanStringFormat = "full"): string {
    if (format === "semantic") return this.toSemanticString();

    // Default "full" format
    const totalDays = this.truncate(TimeSpan.DAY);
    const totalHours = this.truncate(TimeSpan.HOUR);
    const totalMinutes = this.truncate(TimeSpan.MINUTE);
    const totalSeconds = this.truncate(TimeSpan.SECOND);
    const totalMilliseconds = this.truncate(TimeSpan.MILLISECOND);
    const totalMicroseconds = this.truncate(TimeSpan.MICROSECOND);
    const totalNanoseconds = this.truncate(TimeSpan.NANOSECOND);
    const days = totalDays;
    const hours = totalHours.sub(totalDays);
    const minutes = totalMinutes.sub(totalHours);
    const seconds = totalSeconds.sub(totalMinutes);
    const milliseconds = totalMilliseconds.sub(totalSeconds);
    const microseconds = totalMicroseconds.sub(totalMilliseconds);
    const nanoseconds = totalNanoseconds.sub(totalMicroseconds);

    let str = "";
    if (!days.isZero) str += `${days.days}d `;
    if (!hours.isZero) str += `${hours.hours}h `;
    if (!minutes.isZero) str += `${minutes.minutes}m `;
    if (!seconds.isZero) str += `${seconds.seconds}s `;
    if (!milliseconds.isZero) str += `${milliseconds.milliseconds}ms `;
    if (!microseconds.isZero) str += `${microseconds.microseconds}µs `;
    if (!nanoseconds.isZero) str += `${nanoseconds.nanoseconds}ns`;
    return str.trim();
  }

  private toSemanticString(): string {
    const absValue = this.valueOf() < 0n ? -this.valueOf() : this.valueOf();
    const span = new TimeSpan(absValue);
    const isNegative = this.valueOf() < 0n;

    if (span.valueOf() === 0n) return "0s";

    if (span.lessThan(TimeSpan.SECOND)) return "< 1s";

    const totalDays = span.days;
    const totalHours = span.hours;
    const totalMinutes = span.minutes;
    const totalSeconds = span.seconds;

    const years = Math.floor(totalDays / SIMPLE_DAYS_IN_YEAR);
    const months = Math.floor(totalDays / SIMPLE_DAYS_IN_MONTH);
    const weeks = Math.floor(totalDays / 7);
    const days = Math.floor(totalDays);
    const hours = Math.floor(totalHours);
    const minutes = Math.floor(totalMinutes);
    const seconds = Math.floor(totalSeconds);

    const prefix = isNegative ? "-" : "";

    if (years >= 1) {
      let result = `${years}y`;
      if (years < 2) {
        const remainingMonths = Math.floor(
          (totalDays % SIMPLE_DAYS_IN_YEAR) / SIMPLE_DAYS_IN_MONTH,
        );
        if (remainingMonths > 0) result += ` ${remainingMonths}mo`;
      }
      return prefix + result;
    }

    // For durations less than 1 month (30 days), prefer weeks if it's exactly divisible
    if (weeks >= 1 && totalDays < SIMPLE_DAYS_IN_MONTH && totalDays % 7 === 0) {
      let result = `${weeks}w`;
      const remainingDays = Math.floor(totalDays % 7);
      const remainingHoursAfterWeeks = Math.floor(totalHours - weeks * 7 * 24);

      if (weeks < 2)
        if (remainingDays > 0) result += ` ${remainingDays}d`;
        else if (remainingHoursAfterWeeks > 0 && remainingHoursAfterWeeks < 24)
          // Only hours remaining after full weeks (e.g., "1w 1h")
          result += ` ${remainingHoursAfterWeeks}h`;

      return prefix + result;
    }

    if (months >= 1) {
      let result = `${months}mo`;
      if (months < 3) {
        const remainingDays = Math.floor(totalDays % SIMPLE_DAYS_IN_MONTH);
        if (remainingDays > 0) result += ` ${remainingDays}d`;
      }
      return prefix + result;
    }

    if (weeks >= 1) {
      let result = `${weeks}w`;
      const remainingDays = Math.floor(totalDays % 7);
      const remainingHoursAfterWeeks = Math.floor(totalHours - weeks * 7 * 24);

      if (weeks < 2)
        if (remainingDays > 0) result += ` ${remainingDays}d`;
        else if (remainingHoursAfterWeeks > 0 && remainingHoursAfterWeeks < 24)
          // Only hours remaining after full weeks (e.g., "1w 1h")
          result += ` ${remainingHoursAfterWeeks}h`;

      return prefix + result;
    }

    if (days >= 1) {
      let result = `${days}d`;
      const remainingHours = Math.floor(totalHours - days * 24);
      if (days < 2 && remainingHours > 0) result += ` ${remainingHours}h`;
      return prefix + result;
    }

    if (hours >= 1) {
      let result = `${hours}h`;
      if (hours < 3) {
        const remainingMinutes = Math.floor(totalMinutes - hours * 60);
        if (remainingMinutes > 0) result += ` ${remainingMinutes}m`;
      }
      return prefix + result;
    }

    if (minutes >= 1) {
      let result = `${minutes}m`;
      if (minutes < 5) {
        const remainingSeconds = Math.floor(totalSeconds - minutes * 60);
        if (remainingSeconds > 0) result += ` ${remainingSeconds}s`;
      }
      return prefix + result;
    }

    return `${prefix}${seconds}s`;
  }

  /**
   * Multiplies the TimeSpan by a scalar value.
   *
   * @param value - The scalar value to multiply by.
   * @returns A new TimeSpan that is this TimeSpan multiplied by the provided value.
   */
  mult(value: number): TimeSpan {
    return new TimeSpan(math.mult(this.valueOf(), value));
  }

  /**
   * Divides the TimeSpan by a scalar value.
   *
   * @param value - The scalar value to divide by.
   * @returns A new TimeSpan that is this TimeSpan divided by the provided value.
   */
  div(value: number): TimeSpan {
    return new TimeSpan(math.div(this.valueOf(), value));
  }

  /** @returns the decimal number of days in the TimeSpan. */
  get days(): number {
    return Number(this.valueOf()) / Number(TimeSpan.DAY.valueOf());
  }

  /** @returns the decimal number of hours in the TimeSpan. */
  get hours(): number {
    return Number(this.valueOf()) / Number(TimeSpan.HOUR.valueOf());
  }

  /** @returns the decimal number of minutes in the TimeSpan. */
  get minutes(): number {
    return Number(this.valueOf()) / Number(TimeSpan.MINUTE.valueOf());
  }

  /** @returns The number of seconds in the TimeSpan. */
  get seconds(): number {
    return Number(this.valueOf()) / Number(TimeSpan.SECOND.valueOf());
  }

  /** @returns The number of milliseconds in the TimeSpan. */
  get milliseconds(): number {
    return Number(this.valueOf()) / Number(TimeSpan.MILLISECOND.valueOf());
  }

  /** @returns The number of microseconds in the TimeSpan. */
  get microseconds(): number {
    return Number(this.valueOf()) / Number(TimeSpan.MICROSECOND.valueOf());
  }

  /** @returns The number of nanoseconds in the TimeSpan. */
  get nanoseconds(): number {
    return Number(this.valueOf());
  }

  /**
   * Checks if the TimeSpan represents a zero duration.
   *
   * @returns True if the TimeSpan represents a zero duration, false otherwise.
   */
  get isZero(): boolean {
    return this.valueOf() === 0n;
  }

  /**
   * Checks if the TimeSpan is equal to another TimeSpan.
   *
   * @returns True if the TimeSpans are equal, false otherwise.
   */
  equals(other: CrudeTimeSpan): boolean {
    return this.valueOf() === new TimeSpan(other).valueOf();
  }

  /**
   * Adds a TimeSpan to the TimeSpan.
   *
   * @returns A new TimeSpan representing the sum of the two TimeSpans.
   */
  add(other: CrudeTimeSpan): TimeSpan {
    return new TimeSpan(this.valueOf() + new TimeSpan(other).valueOf());
  }

  /**
   * Creates a TimeSpan representing the duration between the two timestamps.
   *
   * @param other
   */
  sub(other: CrudeTimeSpan): TimeSpan {
    return new TimeSpan(this.valueOf() - new TimeSpan(other).valueOf());
  }

  /**
   * Creates a TimeSpan representing the given number of nanoseconds.
   *
   * @param value - The number of nanoseconds.
   * @returns A TimeSpan representing the given number of nanoseconds.
   */
  static nanoseconds(value: math.Numeric = 1): TimeSpan {
    return new TimeSpan(value);
  }

  /** A nanosecond. */
  static readonly NANOSECOND = TimeSpan.nanoseconds(1);

  /**
   * Creates a TimeSpan representing the given number of microseconds.
   *
   * @param value - The number of microseconds.
   * @returns A TimeSpan representing the given number of microseconds.
   */
  static microseconds(value: math.Numeric = 1): TimeSpan {
    return TimeSpan.nanoseconds(math.mult(value, 1000));
  }

  /** A microsecond. */
  static readonly MICROSECOND = TimeSpan.microseconds(1);

  /**
   * Creates a TimeSpan representing the given number of milliseconds.
   *
   * @param value - The number of milliseconds.
   * @returns A TimeSpan representing the given number of milliseconds.
   */
  static milliseconds(value: math.Numeric = 1): TimeSpan {
    return TimeSpan.microseconds(math.mult(value, 1000));
  }

  /** A millisecond. */
  static readonly MILLISECOND = TimeSpan.milliseconds(1);

  /**
   * Creates a TimeSpan representing the given number of seconds.
   *
   * @param value - The number of seconds.
   * @returns A TimeSpan representing the given number of seconds.
   */
  static seconds(value: math.Numeric = 1): TimeSpan {
    return TimeSpan.milliseconds(math.mult(value, 1000));
  }

  /** A second. */
  static readonly SECOND = TimeSpan.seconds(1);

  /**
   * Creates a TimeSpan representing the given number of minutes.
   *
   * @param value - The number of minutes.
   * @returns A TimeSpan representing the given number of minutes.
   */
  static minutes(value: math.Numeric = 1): TimeSpan {
    return TimeSpan.seconds(math.mult(value, 60));
  }

  /** A minute. */
  static readonly MINUTE = TimeSpan.minutes(1);

  /**
   * Creates a TimeSpan representing the given number of hours.
   *
   * @param value - The number of hours.
   * @returns A TimeSpan representing the given number of hours.
   */
  static hours(value: math.Numeric): TimeSpan {
    return TimeSpan.minutes(math.mult(value, 60));
  }

  /** Represents an hour. */
  static readonly HOUR = TimeSpan.hours(1);

  /**
   * Creates a TimeSpan representing the given number of days.
   *
   * @param value - The number of days.
   * @returns A TimeSpan representing the given number of days.
   */
  static days(value: math.Numeric): TimeSpan {
    return TimeSpan.hours(math.mult(value, 24));
  }

  /** Represents a day. */
  static readonly DAY = TimeSpan.days(1);

  /** The maximum possible value for a TimeSpan. */
  static readonly MAX = new TimeSpan(math.MAX_INT64);

  /** The minimum possible value for a TimeSpan. */
  static readonly MIN = new TimeSpan(0);

  /** The zero value for a TimeSpan. */
  static readonly ZERO = new TimeSpan(0);

  /** A zod schema for validating and transforming time spans */
  static readonly z = z.union([
    z.object({ value: z.bigint() }).transform((v) => new TimeSpan(v.value)),
    z.string().transform((n) => new TimeSpan(BigInt(n))),
    z.number().transform((n) => new TimeSpan(n)),
    z.bigint().transform((n) => new TimeSpan(n)),
    z.instanceof(TimeSpan),
    z.instanceof(TimeStamp).transform((t) => new TimeSpan(t)),
    z.custom<Rate>((r) => r instanceof Rate).transform((r) => new TimeSpan(r)),
  ]);
}

/** Rate represents a data rate in Hz. */
export class Rate
  extends primitive.ValueExtension<number>
  implements primitive.Stringer
{
  constructor(value: CrudeRate) {
    if (primitive.isCrudeValueExtension<number>(value)) value = value.value;
    super(value.valueOf());
  }

  /** @returns a pretty string representation of the rate in the format "X Hz". */
  toString(): string {
    return `${this.valueOf()} Hz`;
  }

  /** @returns The number of seconds in the Rate. */
  equals(other: CrudeRate): boolean {
    return this.valueOf() === new Rate(other).valueOf();
  }

  /**
   * Calculates the period of the Rate as a TimeSpan.
   *
   * @returns A TimeSpan representing the period of the Rate.
   */
  get period(): TimeSpan {
    return TimeSpan.seconds(1 / this.valueOf());
  }

  /**
   * Calculates the number of samples in the given TimeSpan at this rate.
   *
   * @param duration - The duration to calculate the sample count from.
   * @returns The number of samples in the given TimeSpan at this rate.
   */
  sampleCount(duration: CrudeTimeSpan): number {
    return new TimeSpan(duration).seconds * this.valueOf();
  }

  /**
   * Calculates the number of bytes in the given TimeSpan at this rate.
   *
   * @param span - The duration to calculate the byte count from.
   * @param density - The density of the data in bytes per sample.
   * @returns The number of bytes in the given TimeSpan at this rate.
   */
  byteCount(span: CrudeTimeSpan, density: CrudeDensity): number {
    return this.sampleCount(span) * new Density(density).valueOf();
  }

  /**
   * Calculates a TimeSpan given the number of samples at this rate.
   *
   * @param sampleCount - The number of samples in the span.
   * @returns A TimeSpan that corresponds to the given number of samples.
   */
  span(sampleCount: number): TimeSpan {
    return TimeSpan.seconds(sampleCount / this.valueOf());
  }

  /**
   * Calculates a TimeSpan given the number of bytes at this rate.
   *
   * @param size - The number of bytes in the span.
   * @param density - The density of the data in bytes per sample.
   * @returns A TimeSpan that corresponds to the given number of bytes.
   */
  byteSpan(size: Size, density: CrudeDensity): TimeSpan {
    return this.span(size.valueOf() / new Density(density).valueOf());
  }

  /**
   * Adds another Rate to this Rate.
   *
   * @param other - The Rate to add.
   * @returns A new Rate representing the sum of the two rates.
   */
  add(other: CrudeRate): Rate {
    return new Rate(math.add(this.valueOf(), new Rate(other).valueOf()));
  }

  /**
   * Subtracts another Rate from this Rate.
   *
   * @param other - The Rate to subtract.
   * @returns A new Rate representing the difference of the two rates.
   */
  sub(other: CrudeRate): Rate {
    return new Rate(math.sub(this.valueOf(), new Rate(other).valueOf()));
  }

  /**
   * Multiplies this Rate by a scalar value.
   *
   * @param value - The scalar value to multiply by.
   * @returns A new Rate representing this Rate multiplied by the value.
   */
  mult(value: number): Rate {
    return new Rate(math.mult(this.valueOf(), value));
  }

  /**
   * Divides this Rate by a scalar value.
   *
   * @param value - The scalar value to divide by.
   * @returns A new Rate representing this Rate divided by the value.
   */
  div(value: number): Rate {
    return new Rate(math.div(this.valueOf(), value));
  }

  /**
   * Creates a Rate representing the given number of Hz.
   *
   * @param value - The number of Hz.
   * @returns A Rate representing the given number of Hz.
   */
  static hz(value: number): Rate {
    return new Rate(value);
  }

  /**
   * Creates a Rate representing the given number of kHz.
   *
   * @param value - The number of kHz.
   * @returns A Rate representing the given number of kHz.
   */
  static khz(value: number): Rate {
    return Rate.hz(value * 1000);
  }

  /** A zod schema for validating and transforming rates */
  static readonly z = z.union([
    z.number().transform((n) => new Rate(n)),
    z.instanceof(Rate),
  ]);
}

/** Density represents the number of bytes in a value. */
export class Density
  extends primitive.ValueExtension<number>
  implements primitive.Stringer
{
  /**
   * Creates a Density representing the given number of bytes per value.
   *
   * @class
   * @param value - The number of bytes per value.
   * @returns A Density representing the given number of bytes per value.
   */
  constructor(value: CrudeDensity) {
    if (primitive.isCrudeValueExtension<number>(value)) value = value.value;
    super(value.valueOf());
  }

  /**
   * Calculates the number of values in the given Size.
   *
   * @param size - The Size to calculate the value count from.
   * @returns The number of values in the given Size.
   */
  length(size: Size): number {
    return size.valueOf() / this.valueOf();
  }

  /**
   * Calculates a Size representing the given number of values.
   *
   * @param sampleCount - The number of values in the Size.
   * @returns A Size representing the given number of values.
   */
  size(sampleCount: number): Size {
    return new Size(sampleCount * this.valueOf());
  }

  /**
   * Adds another Density to this Density.
   *
   * @param other - The Density to add.
   * @returns A new Density representing the sum of the two densities.
   */
  add(other: CrudeDensity): Density {
    return new Density(math.add(this.valueOf(), new Density(other).valueOf()));
  }

  /**
   * Subtracts another Density from this Density.
   *
   * @param other - The Density to subtract.
   * @returns A new Density representing the difference of the two densities.
   */
  sub(other: CrudeDensity): Density {
    return new Density(math.sub(this.valueOf(), new Density(other).valueOf()));
  }

  /**
   * Multiplies this Density by a scalar value.
   *
   * @param value - The scalar value to multiply by.
   * @returns A new Density representing this Density multiplied by the value.
   */
  mult(value: number): Density {
    return new Density(math.mult(this.valueOf(), value));
  }

  /**
   * Divides this Density by a scalar value.
   *
   * @param value - The scalar value to divide by.
   * @returns A new Density representing this Density divided by the value.
   */
  div(value: number): Density {
    return new Density(math.div(this.valueOf(), value));
  }

  /** Unknown/Invalid Density. */
  static readonly UNKNOWN = new Density(0);
  /** 128 bits per value. */
  static readonly BIT128 = new Density(16);
  /** 64 bits per value. */
  static readonly BIT64 = new Density(8);
  /** 32 bits per value. */
  static readonly BIT32 = new Density(4);
  /** 16 bits per value. */
  static readonly BIT16 = new Density(2);
  /** 8 bits per value. */
  static readonly BIT8 = new Density(1);

  /** A zod schema for validating and transforming densities */
  static readonly z = z.union([
    z.number().transform((n) => new Density(n)),
    z.instanceof(Density),
  ]);
}

/**
 * TimeRange is a range of time marked by a start and end timestamp. A TimeRange
 * is start inclusive and end exclusive.
 *
 * @property start - A TimeStamp representing the start of the range.
 * @property end - A Timestamp representing the end of the range.
 */
export class TimeRange implements primitive.Stringer {
  /**
   * The starting TimeStamp of the TimeRange.
   *
   * Note that this value is not guaranteed to be before or equal to the ending value.
   * To ensure that this is the case, call TimeRange.make_valid().
   *
   * In most cases, operations should treat start as inclusive.
   */
  start: TimeStamp;

  /**
   * The starting TimeStamp of the TimeRange.
   *
   * Note that this value is not guaranteed to be before or equal to the ending value.
   * To ensure that this is the case, call TimeRange.make_valid().
   *
   * In most cases, operations should treat end as exclusive.
   */
  end: TimeStamp;

  constructor(tr: CrudeTimeRange);

  constructor(start: CrudeTimeStamp, end: CrudeTimeStamp);

  /**
   * Creates a TimeRange from the given start and end TimeStamps.
   *
   * @param start - A TimeStamp representing the start of the range.
   * @param end - A TimeStamp representing the end of the range.
   */
  constructor(start: CrudeTimeStamp | CrudeTimeRange, end?: CrudeTimeStamp) {
    if (typeof start === "object" && "start" in start) {
      this.start = new TimeStamp(start.start);
      this.end = new TimeStamp(start.end);
    } else {
      this.start = new TimeStamp(start);
      this.end = new TimeStamp(end);
    }
  }

  /** @returns The TimeSpan occupied by the TimeRange. */
  get span(): TimeSpan {
    return new TimeSpan(this.end.valueOf() - this.start.valueOf());
  }

  /**
   * Checks if the timestamp is valid i.e. the start is before the end.
   *
   * @returns True if the TimeRange is valid.
   */
  get isValid(): boolean {
    return this.start.valueOf() <= this.end.valueOf();
  }

  /**
   * Makes sure the TimeRange is valid i.e. the start is before the end.
   *
   * @returns A TimeRange that is valid.
   */
  makeValid(): TimeRange {
    return this.isValid ? this : this.swap();
  }

  /**
   * Checks if the TimeRange is zero (both start and end are TimeStamp.ZERO).
   *
   * @returns True if both start and end are TimeStamp.ZERO, false otherwise.
   */
  get isZero(): boolean {
    return this.start.isZero && this.end.isZero;
  }

  /**
   * @returns the TimeRange as a numeric object with start and end properties.
   */
  get numeric(): NumericTimeRange {
    return { start: Number(this.start.valueOf()), end: Number(this.end.valueOf()) };
  }

  /**
   * Creates a new TimeRange with the start and end swapped.
   *
   * @returns A TimeRange with the start and end swapped.
   */
  swap(): TimeRange {
    return new TimeRange(this.end, this.start);
  }

  get numericBounds(): bounds.Bounds<number> {
    return {
      lower: Number(this.start.valueOf()),
      upper: Number(this.end.valueOf()),
    };
  }

  /**
   * Checks if the TimeRange is equal to the given TimeRange.
   *
   * @param other - The TimeRange to compare to.
   * @returns True if the TimeRange is equal to the given TimeRange.
   */
  equals(other: TimeRange, delta: TimeSpan = TimeSpan.ZERO): boolean {
    if (delta.isZero)
      return this.start.equals(other.start) && this.end.equals(other.end);
    let startDist = this.start.sub(other.start).valueOf();
    let endDist = this.end.sub(other.end).valueOf();
    if (startDist < 0) startDist = -startDist;
    if (endDist < 0) endDist = -endDist;
    return startDist <= delta.valueOf() && endDist <= delta.valueOf();
  }

  /**
   * Returns a string representation of the TimeRange.
   *
   * @returns A string representation of the TimeRange.
   */
  toString(): string {
    return `${this.start.toString()} - ${this.end.toString()}`;
  }

  /**
   * Returns a pretty string representation of the TimeRange.
   *
   * @returns A pretty string representation of the TimeRange.
   */
  toPrettyString(): string {
    return `${this.start.toString("preciseDate")} - ${this.span.toString()}`;
  }

  /**
   * Checks if the two time ranges overlap. If the two time ranges are equal, returns
   * true.  If the start of one range is equal to the end of the other, it returns false.
   * Just follow the rule [start, end), i.e., start is inclusive, and the end is exclusive.
   *
   * @param other - The other TimeRange to compare to.
   * @param delta - A TimeSpan representing the minimum amount of overlap for
   * overlap to return true. This allows for a slight amount of leeway when
   * checking for overlap.
   * @returns True if the two TimeRanges overlap, false otherwise.
   */
  overlapsWith(other: TimeRange, delta: TimeSpan = TimeSpan.ZERO): boolean {
    other = other.makeValid();
    const rng = this.makeValid();

    if (this.equals(other)) return true;

    // If the ranges touch at their boundaries, they do not overlap.
    if (other.end.equals(rng.start) || rng.end.equals(other.start)) return false;

    // Determine the actual overlapping range
    const startOverlap = TimeStamp.max(rng.start, other.start);
    const endOverlap = TimeStamp.min(rng.end, other.end);

    // If the end of overlap is before the start, then they don't overlap at all
    if (endOverlap.before(startOverlap)) return false;

    // Calculate the duration of the overlap
    const overlapDuration = new TimeSpan(endOverlap.sub(startOverlap));

    // Compare the overlap duration with delta
    return overlapDuration.greaterThanOrEqual(delta);
  }

  /**
   * Checks if the TimeRange contains the given TimeRange or TimeStamp.
   *
   * @param other - The TimeRange or TimeStamp to check if it is contained in the TimeRange.
   * @returns True if the TimeRange contains the given TimeRange or TimeStamp.
   */
  contains(other: TimeRange): boolean;

  /**
   * Checks if the TimeRange contains the given TimeStamp.
   *
   * @param ts - The TimeStamp to check if it is contained in the TimeRange.
   * @returns True if the TimeRange contains the given TimeStamp.
   */
  contains(ts: CrudeTimeStamp): boolean;

  contains(other: TimeRange | CrudeTimeStamp): boolean {
    if (other instanceof TimeRange)
      return this.contains(other.start) && this.contains(other.end);
    return this.start.beforeEq(other) && this.end.after(other);
  }

  /**
   * Returns a new TimeRange that is bound by the given TimeRange.
   *
   * @param other - The TimeRange to bound by.
   * @returns A new TimeRange that is bound by the given TimeRange.
   * @example
   * const range = new TimeRange(new TimeStamp(1000), new TimeStamp(2000));
   * const other = new TimeRange(new TimeStamp(1500), new TimeStamp(2500));
   * const bounded = range.boundBy(other);
   * console.log(bounded); // TimeRange(1500, 2000)
   */
  boundBy(other: TimeRange): TimeRange {
    const next = new TimeRange(this.start, this.end);
    if (other.start.after(this.start)) next.start = other.start;
    if (other.start.after(this.end)) next.end = other.start;
    if (other.end.before(this.end)) next.end = other.end;
    if (other.end.before(this.start)) next.start = other.end;
    return next;
  }

  static max(...others: TimeRange[]): TimeRange {
    return new TimeRange(
      TimeStamp.min(...others.map((o) => o.start)),
      TimeStamp.max(...others.map((o) => o.end)),
    );
  }

  /** The maximum possible time range. */
  static readonly MAX = new TimeRange(TimeStamp.MIN, TimeStamp.MAX);

  /** A time range whose start and end are both zero. */
  static readonly ZERO = new TimeRange(TimeStamp.ZERO, TimeStamp.ZERO);

  /** A zod schema for validating and transforming time ranges */
  static readonly z = z.union([
    z
      .object({ start: TimeStamp.z, end: TimeStamp.z })
      .transform((v) => new TimeRange(v.start, v.end)),
    z.instanceof(TimeRange),
  ]);

  /**
   * Sorts two time ranges. The range with the earlier start time is considered less than
   * the range with the later start time. If the start times are equal, the range with the
   * earlier end time is considered less than the range with the later end time.
   *
   * @param a - The first time range.
   * @param b - The second time range.
   * @returns A number indicating the order of the two time ranges. This number is
   * positive if a is earlier than b, negative if a is later than b, and 0 if they are
   * equal.
   */
  static sort(a: TimeRange, b: TimeRange): number {
    return TimeStamp.sort(a.start, b.start) || TimeStamp.sort(a.end, b.end);
  }

  /**
   * Simplify takes the list of `TimeRange`s, makes all of them valid, sorts them, and
   * merges any overlapping ranges.
   *
   * @param ranges - The list of `TimeRange`s to simplify.
   * @returns A list of simplified `TimeRange`s.
   */
  static simplify(ranges: TimeRange[]): TimeRange[] {
    return ranges
      .map((r) => r.makeValid())
      .sort((a, b) => TimeRange.sort(a, b))
      .reduce<TimeRange[]>((simplified, range) => {
        if (range.span.isZero) return simplified;
        if (simplified.length === 0) {
          simplified.push(range);
          return simplified;
        }
        const last = simplified[simplified.length - 1];
        if (last.overlapsWith(range) || last.end.equals(range.start))
          last.end = TimeStamp.max(last.end, range.end);
        else simplified.push(range);
        return simplified;
      }, []);
  }
}

/** DataType is a string that represents a data type. */
export class DataType
  extends primitive.ValueExtension<string>
  implements primitive.Stringer
{
  constructor(value: CrudeDataType) {
    if (primitive.isCrudeValueExtension<string>(value)) value = value.value;
    if (
      value instanceof DataType ||
      typeof value === "string" ||
      typeof value.valueOf() === "string"
    )
      super(value.valueOf() as string);
    else {
      const t = DataType.ARRAY_CONSTRUCTOR_DATA_TYPES.get(value.constructor.name);
      if (t == null)
        throw new Error(`unable to find data type for ${value.toString()}`);
      super(t.valueOf());
    }
  }

  /**
   * @returns the TypedArray constructor for the DataType.
   */
  get Array(): TypedArrayConstructor {
    const v = DataType.ARRAY_CONSTRUCTORS.get(this.toString());
    if (v == null)
      throw new Error(`unable to find array constructor for ${this.valueOf()}`);
    return v;
  }

  /**
   * @returns true if the DataType is equal to the given DataType.
   */
  equals(other: CrudeDataType): boolean {
    return this.valueOf() === other.valueOf();
  }

  /**
   * @returns true if the DataType is equal to any of the given DataTypes.
   */
  matches(...others: CrudeDataType[]): boolean {
    return others.some((o) => this.equals(o));
  }

  /** @returns a string representation of the DataType. If short is true, a 1-4
   * character representation (i64, str, etc.) is returned instead. */
  toString(short: boolean = false): string {
    if (short) return DataType.SHORT_STRINGS.get(this.valueOf()) ?? this.valueOf();
    return this.valueOf();
  }

  /**
   * @returns true if the DataType has a variable density.
   * @example DataType.STRING.isVariable // true
   * @example DataType.INT32.isVariable // false
   */
  get isVariable(): boolean {
    return this.equals(DataType.JSON) || this.equals(DataType.STRING);
  }

  /**
   * @returns true if the DataType is numeric.
   * @example DataType.INT32.isNumeric // true
   * @example DataType.STRING.isNumeric // false
   */
  get isNumeric(): boolean {
    return !this.isVariable && !this.equals(DataType.UUID);
  }

  /**
   * @returns true if the DataType is an integer.
   * @example DataType.INT32.isInteger // true
   * @example DataType.FLOAT32.isInteger // false
   */
  get isInteger(): boolean {
    const str = this.toString();
    return str.startsWith("int") || str.startsWith("uint");
  }

  /**
   * @returns true if the DataType is a floating point number.
   * @example DataType.FLOAT32.isFloat // true
   * @example DataType.INT32.isFloat // false
   */
  get isFloat(): boolean {
    return this.toString().startsWith("float");
  }

  /**
   * @returns the density of the DataType.
   * @example DataType.INT16.density // Density.BIT32
   * @example DataType.FLOAT32.density // Density.BIT32
   */
  get density(): Density {
    const v = DataType.DENSITIES.get(this.toString());
    if (v == null) throw new Error(`unable to find density for ${this.valueOf()}`);
    return v;
  }

  /**
   * @returns true if the DataType is an unsigned integer.
   * @example DataType.UINT32.isUnsigned // true
   * @example DataType.INT32.isUnsigned // false
   */
  get isUnsignedInteger(): boolean {
    return (
      this.equals(DataType.UINT8) ||
      this.equals(DataType.UINT16) ||
      this.equals(DataType.UINT32) ||
      this.equals(DataType.UINT64)
    );
  }

  /**
   * @returns true if the DataType is a signed integer.
   * @example DataType.INT32.isSigned // true
   * @example DataType.UINT32.isSigned // false
   */
  get isSignedInteger(): boolean {
    return (
      this.equals(DataType.INT8) ||
      this.equals(DataType.INT16) ||
      this.equals(DataType.INT32) ||
      this.equals(DataType.INT64)
    );
  }

  /** @returns true if the data type can be cast to the other data type without loss of precision. */
  canSafelyCastTo(other: DataType): boolean {
    if (this.equals(other)) return true;
    if (!this.isNumeric || !other.isNumeric) return false;
    if (this.isVariable || other.isVariable) return false;
    if (this.isUnsignedInteger && other.isSignedInteger) return false;

    if (this.isFloat)
      return other.isFloat && this.density.valueOf() <= other.density.valueOf();
    if (this.equals(DataType.INT32) && other.equals(DataType.FLOAT64)) return true;
    if (this.equals(DataType.INT8) && other.equals(DataType.FLOAT32)) return true;
    if (this.isInteger && other.isInteger)
      return (
        this.density.valueOf() <= other.density.valueOf() &&
        this.isUnsignedInteger === other.isUnsignedInteger
      );

    return false;
  }

  /** @returns true if the data type can be cast to the other data type, even if there is a loss of precision. */
  canCastTo(other: DataType): boolean {
    if (this.isNumeric && other.isNumeric) return true;
    return this.equals(other);
  }

  /**
   * Checks whether the given TypedArray is of the same type as the DataType.
   *
   * @param array - The TypedArray to check.
   * @returns True if the TypedArray is of the same type as the DataType.
   */
  checkArray(array: TypedArray): boolean {
    return array.constructor === this.Array;
  }

  /** @returns true if the data type uses bigints to store values. */
  get usesBigInt(): boolean {
    return DataType.BIG_INT_TYPES.some((t) => t.equals(this));
  }

  /** Represents an Unknown/Invalid DataType. */
  static readonly UNKNOWN = new DataType("unknown");
  /** Represents a 64-bit floating point value. */
  static readonly FLOAT64 = new DataType("float64");
  /** Represents a 32-bit floating point value. */
  static readonly FLOAT32 = new DataType("float32");
  /** Represents a 64-bit signed integer value. */
  static readonly INT64 = new DataType("int64");
  /** Represents a 32-bit signed integer value. */
  static readonly INT32 = new DataType("int32");
  /** Represents a 16-bit signed integer value. */
  static readonly INT16 = new DataType("int16");
  /** Represents a 8-bit signed integer value. */
  static readonly INT8 = new DataType("int8");
  /** Represents a 64-bit unsigned integer value. */
  static readonly UINT64 = new DataType("uint64");
  /** Represents a 32-bit unsigned integer value. */
  static readonly UINT32 = new DataType("uint32");
  /** Represents a 16-bit unsigned integer value. */
  static readonly UINT16 = new DataType("uint16");
  /** Represents a 8-bit unsigned integer value. */
  static readonly UINT8 = new DataType("uint8");
  /** Represents a boolean value. Stored as a 8-bit unsigned integer. */
  static readonly BOOLEAN = new DataType("boolean");
  /** Represents a 64-bit unix epoch. */
  static readonly TIMESTAMP = new DataType("timestamp");
  /** Represents a UUID data type. */
  static readonly UUID = new DataType("uuid");
  /** Represents a string data type. Strings have an unknown density, and are separate
   * by a newline character. */
  static readonly STRING = new DataType("string");
  /** Represents a JSON data type. JSON has an unknown density, and is separated by a
   * newline character. */
  static readonly JSON = new DataType("json");

  private static readonly ARRAY_CONSTRUCTORS: Map<string, TypedArrayConstructor> =
    new Map<string, TypedArrayConstructor>([
      [DataType.UINT8.toString(), Uint8Array],
      [DataType.UINT16.toString(), Uint16Array],
      [DataType.UINT32.toString(), Uint32Array],
      [DataType.UINT64.toString(), BigUint64Array],
      [DataType.FLOAT32.toString(), Float32Array],
      [DataType.FLOAT64.toString(), Float64Array],
      [DataType.INT8.toString(), Int8Array],
      [DataType.INT16.toString(), Int16Array],
      [DataType.INT32.toString(), Int32Array],
      [DataType.INT64.toString(), BigInt64Array],
      [DataType.TIMESTAMP.toString(), BigInt64Array],
      [DataType.STRING.toString(), Uint8Array],
      [DataType.JSON.toString(), Uint8Array],
      [DataType.UUID.toString(), Uint8Array],
    ]);

  private static readonly ARRAY_CONSTRUCTOR_DATA_TYPES: Map<string, DataType> = new Map<
    string,
    DataType
  >([
    [Uint8Array.name, DataType.UINT8],
    [Uint16Array.name, DataType.UINT16],
    [Uint32Array.name, DataType.UINT32],
    [BigUint64Array.name, DataType.UINT64],
    [Float32Array.name, DataType.FLOAT32],
    [Float64Array.name, DataType.FLOAT64],
    [Int8Array.name, DataType.INT8],
    [Int16Array.name, DataType.INT16],
    [Int32Array.name, DataType.INT32],
    [BigInt64Array.name, DataType.INT64],
  ]);

  private static readonly DENSITIES = new Map<string, Density>([
    [DataType.UINT8.toString(), Density.BIT8],
    [DataType.UINT16.toString(), Density.BIT16],
    [DataType.UINT32.toString(), Density.BIT32],
    [DataType.UINT64.toString(), Density.BIT64],
    [DataType.FLOAT32.toString(), Density.BIT32],
    [DataType.FLOAT64.toString(), Density.BIT64],
    [DataType.INT8.toString(), Density.BIT8],
    [DataType.INT16.toString(), Density.BIT16],
    [DataType.INT32.toString(), Density.BIT32],
    [DataType.INT64.toString(), Density.BIT64],
    [DataType.TIMESTAMP.toString(), Density.BIT64],
    [DataType.STRING.toString(), Density.UNKNOWN],
    [DataType.JSON.toString(), Density.UNKNOWN],
    [DataType.UUID.toString(), Density.BIT128],
  ]);

  /** All the data types. */
  static readonly ALL = [
    DataType.UNKNOWN,
    DataType.FLOAT64,
    DataType.FLOAT32,
    DataType.INT64,
    DataType.INT32,
    DataType.INT16,
    DataType.INT8,
    DataType.UINT64,
    DataType.UINT32,
    DataType.UINT16,
    DataType.UINT8,
    DataType.TIMESTAMP,
    DataType.UUID,
    DataType.STRING,
    DataType.JSON,
  ];

  private static readonly SHORT_STRINGS = new Map<string, string>([
    [DataType.UINT8.toString(), "u8"],
    [DataType.UINT16.toString(), "u16"],
    [DataType.UINT32.toString(), "u32"],
    [DataType.UINT64.toString(), "u64"],
    [DataType.INT8.toString(), "i8"],
    [DataType.INT16.toString(), "i16"],
    [DataType.INT32.toString(), "i32"],
    [DataType.INT64.toString(), "i64"],
    [DataType.FLOAT32.toString(), "f32"],
    [DataType.FLOAT64.toString(), "f64"],
    [DataType.BOOLEAN.toString(), "bool"],
    [DataType.TIMESTAMP.toString(), "ts"],
    [DataType.UUID.toString(), "uuid"],
    [DataType.STRING.toString(), "str"],
    [DataType.JSON.toString(), "json"],
  ]);

  static readonly BIG_INT_TYPES = [DataType.INT64, DataType.UINT64, DataType.TIMESTAMP];

  /** A zod schema for a DataType. */
  static readonly z = z.union([
    z.string().transform((v) => new DataType(v)),
    z.instanceof(DataType),
  ]);
}

/**
 * The Size of an element in bytes.
 */
export class Size
  extends primitive.ValueExtension<number>
  implements primitive.Stringer
{
  constructor(value: CrudeSize) {
    if (primitive.isCrudeValueExtension<number>(value)) value = value.value;
    super(value.valueOf());
  }

  /** @returns true if the Size is larger than the other size. */
  largerThan(other: CrudeSize): boolean {
    if (primitive.isCrudeValueExtension<number>(other)) other = other.value;
    return this.valueOf() > other.valueOf();
  }

  /** @returns true if the Size is smaller than the other size. */
  smallerThan(other: CrudeSize): boolean {
    if (primitive.isCrudeValueExtension<number>(other)) other = other.value;
    return this.valueOf() < other.valueOf();
  }

  /** @returns a new Size representing the sum of the two Sizes. */
  add(other: CrudeSize): Size {
    if (primitive.isCrudeValueExtension<number>(other)) other = other.value;
    return new Size(math.add(this.valueOf(), other.valueOf()));
  }

  /** @returns a new Size representing the difference of the two Sizes. */
  sub(other: CrudeSize): Size {
    if (primitive.isCrudeValueExtension<number>(other)) other = other.value;
    return new Size(math.sub(this.valueOf(), other.valueOf()));
  }

  /**
   * Multiplies this Size by a scalar value.
   *
   * @param value - The scalar value to multiply by.
   * @returns A new Size representing this Size multiplied by the value.
   */
  mult(value: number): Size {
    return new Size(math.mult(this.valueOf(), value));
  }

  /**
   * Divides this Size by a scalar value.
   *
   * @param value - The scalar value to divide by.
   * @returns A new Size representing this Size divided by the value.
   */
  div(value: number): Size {
    return new Size(math.div(this.valueOf(), value));
  }

  /** @returns a new Size representing the truncated value of the Size. */
  truncate(span: CrudeSize): Size {
    return new Size(
      Math.trunc(this.valueOf() / new Size(span).valueOf()) * new Size(span).valueOf(),
    );
  }

  /** @returns a new Size representing the remainder of the Size. */
  remainder(span: CrudeSize): Size {
    return Size.bytes(this.valueOf() % new Size(span).valueOf());
  }

  /** @returns the number of gigabytes in the Size. */
  get gigabytes(): number {
    return this.valueOf() / Size.GIGABYTE.valueOf();
  }

  /** @returns the number of megabytes in the Size. */
  get megabytes(): number {
    return this.valueOf() / Size.MEGABYTE.valueOf();
  }

  /** @returns the number of kilobytes in the Size. */
  get kilobytes(): number {
    return this.valueOf() / Size.KILOBYTE.valueOf();
  }

  /** @returns the number of terabytes in the Size. */
  get terabytes(): number {
    return this.valueOf() / Size.TERABYTE.valueOf();
  }

  /** @returns a nicely formatted string representation of the Size. */
  toString(): string {
    const totalTB = this.truncate(Size.TERABYTE);
    const totalGB = this.truncate(Size.GIGABYTE);
    const totalMB = this.truncate(Size.MEGABYTE);
    const totalKB = this.truncate(Size.KILOBYTE);
    const totalB = this.truncate(Size.BYTE);
    const tb = totalTB;
    const gb = totalGB.sub(totalTB);
    const mb = totalMB.sub(totalGB);
    const kb = totalKB.sub(totalMB);
    const bytes = totalB.sub(totalKB);
    let str = "";
    if (!tb.isZero) str += `${tb.terabytes}TB `;
    if (!gb.isZero) str += `${gb.gigabytes}GB `;
    if (!mb.isZero) str += `${mb.megabytes}MB `;
    if (!kb.isZero) str += `${kb.kilobytes}KB `;
    if (!bytes.isZero || str === "") str += `${bytes.valueOf()}B`;
    return str.trim();
  }

  /**
   * Creates a Size from the given number of bytes.
   *
   * @param value - The number of bytes.
   * @returns A Size representing the given number of bytes.
   */
  static bytes(value: CrudeSize = 1): Size {
    return new Size(value);
  }

  /** A single byte */
  static readonly BYTE = new Size(1);

  /**
   * Creates a Size from the given number if kilobytes.
   *
   * @param value - The number of kilobytes.
   * @returns A Size representing the given number of kilobytes.
   */
  static kilobytes(value: CrudeSize = 1): Size {
    return Size.bytes(new Size(value).valueOf() * 1e3);
  }

  /** A kilobyte */
  static readonly KILOBYTE = Size.kilobytes(1);

  /**
   * Creates a Size from the given number of megabytes.
   *
   * @param value - The number of megabytes.
   * @returns A Size representing the given number of megabytes.
   */
  static megabytes(value: CrudeSize = 1): Size {
    return Size.kilobytes(new Size(value).valueOf() * 1e3);
  }

  /** A megabyte */
  static readonly MEGABYTE = Size.megabytes(1);

  /**
   * Creates a Size from the given number of gigabytes.
   *
   * @param value - The number of gigabytes.
   * @returns A Size representing the given number of gigabytes.
   */
  static gigabytes(value: CrudeSize = 1): Size {
    return Size.megabytes(new Size(value).valueOf() * 1e3);
  }

  /** A gigabyte */
  static readonly GIGABYTE = Size.gigabytes(1);

  /**
   * Creates a Size from the given number of terabytes.
   *
   * @param value - The number of terabytes.
   * @returns  A Size representing the given number of terabytes.
   */
  static terabytes(value: CrudeSize): Size {
    return Size.gigabytes(new Size(value).valueOf() * 1e3);
  }

  /** A terabyte. */
  static readonly TERABYTE = Size.terabytes(1);

  /** The zero value for Size */
  static readonly ZERO = new Size(0);

  /** A zod schema for a Size. */
  static readonly z = z.union([
    z.number().transform((v) => new Size(v)),
    z.instanceof(Size),
  ]);

  /** @returns true if the Size is zero. */
  get isZero(): boolean {
    return this.valueOf() === 0;
  }
}

export type CrudeTimeStamp =
  | bigint
  | TimeStamp
  | TimeSpan
  | number
  | Date
  | string
  | DateComponents
  | primitive.CrudeValueExtension<bigint>;
export type TimeStampT = number;
export type CrudeTimeSpan =
  | bigint
  | TimeSpan
  | TimeStamp
  | number
  | Rate
  | primitive.CrudeValueExtension<bigint>;
export type TimeSpanT = number;
export type CrudeRate = Rate | number | primitive.CrudeValueExtension<number>;
export type RateT = number;
export type CrudeDensity = Density | number | primitive.CrudeValueExtension<number>;
export type DensityT = number;
export type CrudeDataType =
  | DataType
  | string
  | TypedArray
  | primitive.CrudeValueExtension<string>;
export type DataTypeT = string;
export type CrudeSize = Size | number | primitive.CrudeValueExtension<number>;
export type SizeT = number;
export interface CrudeTimeRange {
  start: CrudeTimeStamp;
  end: CrudeTimeStamp;
}

export const numericTimeRangeZ = z.object({
  start: z.number(),
  end: z.number(),
});

/**
 * A time range backed by numbers instead of TimeStamps/BigInts.
 * Involves a loss of precision, but can be useful for serialization.
 */
export interface NumericTimeRange extends z.infer<typeof numericTimeRangeZ> {}

export const typedArrayZ = z.union([
  z.instanceof(Uint8Array),
  z.instanceof(Uint16Array),
  z.instanceof(Uint32Array),
  z.instanceof(BigUint64Array),
  z.instanceof(Float32Array),
  z.instanceof(Float64Array),
  z.instanceof(Int8Array),
  z.instanceof(Int16Array),
  z.instanceof(Int32Array),
  z.instanceof(BigInt64Array),
]);

export type TypedArray = z.infer<typeof typedArrayZ>;

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
export type TelemValue =
  | number
  | bigint
  | string
  | boolean
  | Date
  | TimeStamp
  | TimeSpan;

export const isTelemValue = (value: unknown): value is TelemValue => {
  const ot = typeof value;
  return (
    ot === "string" ||
    ot === "number" ||
    ot === "boolean" ||
    ot === "bigint" ||
    value instanceof TimeStamp ||
    value instanceof TimeSpan ||
    value instanceof Date
  );
};

export const convertDataType = (
  source: DataType,
  target: DataType,
  value: math.Numeric,
  offset: math.Numeric = 0,
): math.Numeric => {
  if (source.usesBigInt && !target.usesBigInt) return Number(value) - Number(offset);
  if (!source.usesBigInt && target.usesBigInt)
    return BigInt(value.valueOf()) - BigInt(offset.valueOf());
  return math.sub(value, offset);
};
