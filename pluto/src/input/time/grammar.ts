// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

/** The instants a typed expression may be anchored on. */
export interface Anchors {
  now: TimeStamp;
  /** The other end of the range the cell belongs to. */
  start?: TimeStamp;
  end?: TimeStamp;
  /** The parent range's start, the `T` in `T+3.2s`. */
  parent?: TimeStamp;
}

const SPAN_UNITS: Record<string, TimeSpan> = {
  d: TimeSpan.DAY,
  h: TimeSpan.HOUR,
  m: TimeSpan.MINUTE,
  min: TimeSpan.MINUTE,
  s: TimeSpan.SECOND,
  ms: TimeSpan.MILLISECOND,
  us: TimeSpan.MICROSECOND,
  µs: TimeSpan.MICROSECOND,
  ns: TimeSpan.NANOSECOND,
};

const UNIT_RE = /^(\d+(?:\.\d+)?)\s*(ms|µs|us|ns|min|d|h|m|s)\s*/i;
const CLOCK_RE = /^(\d+):(\d{1,2})(?::(\d{1,2})(?:\.(\d{1,9}))?)?$/;
const BARE_NUMBER_RE = /^\d+(?:\.\d+)?$/;

const fractionToNanoseconds = (fraction: string | undefined): bigint =>
  fraction == null ? 0n : BigInt(fraction.padEnd(9, "0").slice(0, 9));

/**
 * Parses a duration. Accepts unit runs ("30s", "2h 30m", "1.5h", "500ms", "250us"),
 * clock form ("1:30:00", "0:30", "00:00:30.250"), and a bare number of seconds.
 * @returns the span, or null when the text is not a duration.
 */
export const parseTimeSpan = (text: string): TimeSpan | null => {
  let rest = text.trim();
  if (rest.length === 0) return null;
  const clock = CLOCK_RE.exec(rest);
  if (clock != null) {
    const [, hours, minutes, seconds, fraction] = clock;
    return TimeSpan.hours(Number(hours))
      .add(TimeSpan.minutes(Number(minutes)))
      .add(TimeSpan.seconds(Number(seconds ?? 0)))
      .add(new TimeSpan(fractionToNanoseconds(fraction)));
  }
  if (BARE_NUMBER_RE.test(rest)) return TimeSpan.SECOND.mult(Number(rest));
  let total = TimeSpan.ZERO;
  while (rest.length > 0) {
    const match = UNIT_RE.exec(rest);
    if (match == null) return null;
    const [whole, amount, unit] = match;
    total = total.add(SPAN_UNITS[unit.toLowerCase()].mult(Number(amount)));
    rest = rest.slice(whole.length);
  }
  return total;
};

const ANCHOR_RE = /^(now|start|end|t)\s*(?:([+-])\s*(.+))?$/i;
const DATE_RE = /^(\d{4})-(\d{2})-(\d{2})$/;
const TIME_RE = /^(\d{1,2}):(\d{2})(?::(\d{2})(?:\.(\d{1,9}))?)?\s*(am|pm)?$/i;
const EPOCH_RE = /^\d{10,19}$/;

interface Clock {
  hour: number;
  minute: number;
  second: number;
  fraction: bigint;
}

const parseClock = (text: string): Clock | null => {
  const match = TIME_RE.exec(text);
  if (match == null) return null;
  const [, h, m, s, fraction, meridiem] = match;
  let hour = Number(h);
  if (meridiem != null) {
    if (hour > 12) return null;
    hour %= 12;
    if (meridiem.toLowerCase() === "pm") hour += 12;
  }
  return {
    hour,
    minute: Number(m),
    second: Number(s ?? 0),
    fraction: fractionToNanoseconds(fraction),
  };
};

const fromLocalDate = (date: Date, fraction: bigint): TimeStamp =>
  new TimeStamp(date).add(new TimeSpan(fraction));

const parseAbsolute = (text: string, now: TimeStamp): TimeStamp | null => {
  if (EPOCH_RE.test(text)) {
    if (text.length <= 10) return new TimeStamp(TimeSpan.seconds(Number(text)));
    if (text.length <= 13) return new TimeStamp(TimeSpan.milliseconds(Number(text)));
    return new TimeStamp(BigInt(text));
  }
  // A full ISO string with a zone is unambiguous; let Date take it.
  if (/^\d{4}-\d{2}-\d{2}T.*(Z|[+-]\d{2}:?\d{2})$/.test(text)) {
    const fraction = /\.(\d{4,9})/.exec(text)?.[1];
    const ms = Date.parse(
      text.replace(/\.(\d{4,9})/, (_, f: string) => `.${f.slice(0, 3)}`),
    );
    if (Number.isNaN(ms)) return null;
    const sub = fraction == null ? 0n : fractionToNanoseconds(fraction) % 1000000n;
    return fromLocalDate(new Date(ms), sub);
  }
  const [datePart, ...timeParts] = text.split(/[T ]/);
  const date = DATE_RE.exec(datePart);
  if (date != null) {
    const [, y, mo, d] = date;
    const local = new Date(Number(y), Number(mo) - 1, Number(d));
    if (timeParts.length === 0) return fromLocalDate(local, 0n);
    const clock = parseClock(timeParts.join(" "));
    if (clock == null) return null;
    local.setHours(clock.hour, clock.minute, clock.second, 0);
    return fromLocalDate(local, clock.fraction);
  }
  const clock = parseClock(text);
  if (clock == null) return null;
  const today = new Date(Number(now.valueOf() / 1000000n));
  today.setHours(clock.hour, clock.minute, clock.second, 0);
  return fromLocalDate(today, clock.fraction);
};

/** Why a typed expression did not resolve. */
export type ParseFailure = "invalid" | "no-parent" | "no-anchor";

export type ParseResult =
  | { ok: true; value: TimeStamp; anchor?: keyof Anchors }
  | { ok: false; reason: ParseFailure };

/**
 * Parses an instant: an anchor word (`now`, `start`, `end`, `T`) with an optional
 * signed duration, an absolute local date-time (`2026-08-23 14:05:00.250`), a time of
 * day meaning today (`14:05`, `2:05 pm`), an ISO string with a zone, or an epoch in
 * seconds, milliseconds, or nanoseconds.
 */
export const parseTimeStamp = (text: string, anchors: Anchors): ParseResult => {
  // The formatted layout groups fractional digits with spaces; accept them back.
  const trimmed = text.trim().replace(/(?<=\.\d{3}(?: \d{3})?) (?=\d{3})/g, "");
  if (trimmed.length === 0) return { ok: false, reason: "invalid" };
  const anchored = ANCHOR_RE.exec(trimmed);
  if (anchored != null) {
    const [, word, sign, spanText] = anchored;
    const key = word.toLowerCase();
    const name: keyof Anchors = key === "t" ? "parent" : (key as keyof Anchors);
    const base = anchors[name];
    if (base == null)
      return { ok: false, reason: name === "parent" ? "no-parent" : "no-anchor" };
    if (sign == null) return { ok: true, value: base, anchor: name };
    const span = parseTimeSpan(spanText);
    if (span == null) return { ok: false, reason: "invalid" };
    const value = sign === "+" ? base.add(span) : base.sub(span);
    return { ok: true, value, anchor: name };
  }
  const value = parseAbsolute(trimmed, anchors.now);
  if (value == null) return { ok: false, reason: "invalid" };
  return { ok: true, value };
};

const pad = (n: number, width: number): string => n.toString().padStart(width, "0");

/**
 * Rounds a form value (nanoseconds as a float64) to the microsecond: a float holds
 * about 0.25 us at today's epoch, so finer digits are noise.
 */
export const roundNumeric = (value: number): bigint => {
  const ns = BigInt(Math.trunc(value));
  return ((ns + 500n) / 1000n) * 1000n;
};

/** Converts a form value to an instant, rounded by {@link roundNumeric}. */
export const fromNumeric = (value: number): TimeStamp =>
  new TimeStamp(roundNumeric(value));

/**
 * Formats an instant in local time as `YYYY-MM-DD HH:MM:SS`, followed by the
 * fractional second in three-digit groups (`.250 137 004`) with trailing zero groups
 * dropped. The layout is fixed so a caret position names a unit.
 */
export const formatTimeStamp = (ts: TimeStamp): string => {
  const ns = ts.valueOf();
  const date = new Date(Number(ns / 1000000n));
  const base = `${date.getFullYear()}-${pad(date.getMonth() + 1, 2)}-${pad(
    date.getDate(),
    2,
  )} ${pad(date.getHours(), 2)}:${pad(date.getMinutes(), 2)}:${pad(date.getSeconds(), 2)}`;
  const subSecond = ((ns % 1000000000n) + 1000000000n) % 1000000000n;
  if (subSecond === 0n) return base;
  const digits = subSecond.toString().padStart(9, "0");
  const groups = [digits.slice(0, 3), digits.slice(3, 6), digits.slice(6, 9)];
  while (groups.length > 0 && groups[groups.length - 1] === "000") groups.pop();
  return `${base}.${groups.join(" ")}`;
};

const DAY_MS = 86_400_000;

const localMidnight = (date: Date): number =>
  new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();

/** The instant as a Date, to the millisecond. */
export const toDate = (ts: TimeStamp): Date =>
  new Date(Number(ts.valueOf() / 1000000n));

/**
 * Names the day of `ts` the way a person would relative to `now`: `Today`,
 * `Yesterday`, `Tomorrow`, a weekday within six days, else `Aug 21`, with the year
 * appended only when it differs from the current one.
 */
export const describeDay = (ts: TimeStamp, now: TimeStamp): string => {
  const date = toDate(ts);
  const days = Math.round((localMidnight(date) - localMidnight(toDate(now))) / DAY_MS);
  if (days === 0) return "Today";
  if (days === -1) return "Yesterday";
  if (days === 1) return "Tomorrow";
  if (Math.abs(days) <= 6)
    return date.toLocaleDateString(undefined, { weekday: "long" });
  const sameYear = date.getFullYear() === toDate(now).getFullYear();
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: sameYear ? undefined : "numeric",
  });
};

/** Whether two instants fall on the same local calendar day. */
export const sameDay = (a: TimeStamp, b: TimeStamp): boolean =>
  localMidnight(toDate(a)) === localMidnight(toDate(b));

/** Drops the digits of a span below `resolution`. */
export const truncateSpan = (span: TimeSpan, resolution: TimeSpan): TimeSpan =>
  new TimeSpan((span.valueOf() / resolution.valueOf()) * resolution.valueOf());

/**
 * The time of day: `14:05`, then `:32` when seconds are set, then the fractional
 * groups of {@link formatTimeStamp} when they are set. Digits below `resolution` are
 * dropped first.
 */
export const formatTime = (ts: TimeStamp, resolution?: TimeSpan): string => {
  const shown = resolution == null ? ts : ts.truncate(resolution);
  const full = formatTimeStamp(shown).slice(11);
  return full.endsWith(":00") ? full.slice(0, 5) : full;
};

/** A unit of the fixed layout, in caret order. */
export type Unit =
  | "year"
  | "month"
  | "day"
  | "hour"
  | "minute"
  | "second"
  | "millisecond"
  | "microsecond"
  | "nanosecond";

const UNIT_BOUNDARIES: [number, Unit][] = [
  [4, "year"],
  [7, "month"],
  [10, "day"],
  [13, "hour"],
  [16, "minute"],
  [19, "second"],
  [23, "millisecond"],
  [27, "microsecond"],
  [31, "nanosecond"],
];

/** Names the unit the caret sits in for a string produced by {@link formatTimeStamp}. */
export const unitAt = (caret: number): Unit => {
  for (const [end, unit] of UNIT_BOUNDARIES) if (caret <= end) return unit;
  return "nanosecond";
};

const UNIT_SPANS: Partial<Record<Unit, TimeSpan>> = {
  day: TimeSpan.DAY,
  hour: TimeSpan.HOUR,
  minute: TimeSpan.MINUTE,
  second: TimeSpan.SECOND,
  millisecond: TimeSpan.MILLISECOND,
  microsecond: TimeSpan.MICROSECOND,
  nanosecond: TimeSpan.NANOSECOND,
};

/**
 * Moves an instant by `steps` of `unit`. Months and years move on the local calendar
 * and keep the sub-millisecond remainder; every other unit is a fixed span.
 */
export const nudge = (ts: TimeStamp, unit: Unit, steps: number): TimeStamp => {
  const span = UNIT_SPANS[unit];
  if (span != null) return ts.add(span.mult(steps));
  const ns = ts.valueOf();
  const date = new Date(Number(ns / 1000000n));
  const subMs = ns % 1000000n;
  if (unit === "year") date.setFullYear(date.getFullYear() + steps);
  else date.setMonth(date.getMonth() + steps);
  return new TimeStamp(date).add(new TimeSpan(subMs));
};

/** The local zone's short name for an instant, e.g. "PDT". */
export const zoneName = (ts: TimeStamp): string => {
  const date = new Date(Number(ts.valueOf() / 1000000n));
  const part = new Intl.DateTimeFormat(undefined, { timeZoneName: "short" })
    .formatToParts(date)
    .find((p) => p.type === "timeZoneName");
  return part?.value ?? "";
};
