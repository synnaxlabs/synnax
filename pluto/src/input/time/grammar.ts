// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { Text as TelemText } from "@/telem/text";

/** The instants a typed expression may be anchored on. */
export interface Anchors {
  now: TimeStamp;
  /** The other end of the range the cell belongs to. */
  start?: TimeStamp;
  end?: TimeStamp;
  /** The parent range's start, the `T` in `T+3.2s`. */
  parent?: TimeStamp;
}

const CLOCK_RE = /^(\d+):(\d{1,2})(?::(\d{1,2})(?:\.(\d{1,9}))?)?$/;
const BARE_NUMBER_RE = /^\d+(?:\.\d+)?$/;

const fractionToNanoseconds = (fraction: string | undefined): bigint =>
  fraction == null ? 0n : BigInt(fraction.padEnd(9, "0").slice(0, 9));

export interface ParsedTimeSpan {
  value: TimeSpan;
  /** The form the duration was typed in. */
  kind: "units" | "clock" | "seconds";
}

/**
 * Parses a duration. Accepts unit runs ("30s", "2h 30m", "1.5h", "500ms", "250us"),
 * clock form ("1:30:00", "0:30", "00:00:30.250"), and a bare number of seconds.
 * @returns the span and the form it took, or null when the text is not a duration.
 */
export const parseTimeSpan = (text: string): ParsedTimeSpan | null => {
  const trimmed = text.trim();
  const clock = CLOCK_RE.exec(trimmed);
  if (clock != null) {
    const [, hours, minutes, seconds, fraction] = clock;
    const value = TimeSpan.hours(Number(hours))
      .add(TimeSpan.minutes(Number(minutes)))
      .add(TimeSpan.seconds(Number(seconds ?? 0)))
      .add(new TimeSpan(fractionToNanoseconds(fraction)));
    return { value, kind: "clock" };
  }
  if (BARE_NUMBER_RE.test(trimmed))
    return { value: TimeSpan.SECOND.mult(Number(trimmed)), kind: "seconds" };
  const value = TimeSpan.parse(trimmed);
  return value == null ? null : { value, kind: "units" };
};

const ANCHOR_RE = /^(now|start|end|t)\s*(?:([+-])\s*(.+))?$/i;
const EPOCH_RE = /^\d{9,19}$/;
const ZONE_RE = /T.*(Z|[+-]\d{2}:?\d{2})$/;
const TIME_OF_DAY_RE = /^\d{1,2}:\d{2}/;

type EpochUnit = "seconds" | "milliseconds" | "microseconds" | "nanoseconds";

const EPOCH_UNITS: [EpochUnit, TimeSpan][] = [
  ["seconds", TimeSpan.SECOND],
  ["milliseconds", TimeSpan.MILLISECOND],
  ["microseconds", TimeSpan.MICROSECOND],
  ["nanoseconds", TimeSpan.NANOSECOND],
];

const EPOCH_MIN = new TimeStamp(new Date(1980, 0, 1));
const EPOCH_MAX = new TimeStamp(new Date(2100, 0, 1));

/**
 * Reads a run of digits as an epoch. The units sit a factor of 1000 apart and the
 * plausible window spans far less, so at most one unit lands inside it. Digits no unit
 * places there are nanoseconds.
 */
const parseEpoch = (digits: string): ParsedTimeStamp => {
  const n = BigInt(digits);
  for (const [unit, span] of EPOCH_UNITS) {
    const value = new TimeStamp(n * span.valueOf());
    if (value.before(EPOCH_MIN) || value.after(EPOCH_MAX)) continue;
    return { value, kind: "epoch", unit };
  }
  return { value: new TimeStamp(n), kind: "epoch", unit: "nanoseconds" };
};

/**
 * How an expression was read. An anchor's `offset` is signed, and zero for the bare
 * word. `clock` is a time of day alone, placed on today; `local` is a local date with
 * or without a time of day.
 */
export type Reading =
  | { kind: "anchor"; anchor: keyof Anchors; offset: TimeSpan }
  | { kind: "epoch"; unit: EpochUnit }
  | { kind: "iso" }
  | { kind: "clock" }
  | { kind: "local" };

type ParsedTimeStamp = { value: TimeStamp } & Reading;

const parseAbsolute = (text: string, now: TimeStamp): ParsedTimeStamp | null => {
  if (EPOCH_RE.test(text)) return parseEpoch(text);
  if (TIME_OF_DAY_RE.test(text)) {
    const today = now.toPreciseString("local").slice(0, 10);
    const value = TimeStamp.parse(`${today} ${text}`, "local");
    return value == null ? null : { value, kind: "clock" };
  }
  const value = TimeStamp.parse(text, "local");
  if (value == null) return null;
  return { value, kind: ZONE_RE.test(text) ? "iso" : "local" };
};

/** Why a typed expression did not resolve. */
export type ParseFailure = "invalid" | "no-parent" | "no-anchor";

export type ParseResult =
  ({ ok: true } & ParsedTimeStamp) | { ok: false; reason: ParseFailure };

/**
 * Parses an instant: an anchor word (`now`, `start`, `end`, `T`) with an optional
 * signed duration, an absolute local date-time (`2026-08-23 14:05:00.250`), a time of
 * day meaning today (`14:05`, `2:05 pm`), an ISO string with a zone, or an epoch in
 * seconds, milliseconds, or nanoseconds.
 */
export const parseTimeStamp = (text: string, anchors: Anchors): ParseResult => {
  const trimmed = text.trim();
  if (trimmed.length === 0) return { ok: false, reason: "invalid" };
  const anchored = ANCHOR_RE.exec(trimmed);
  if (anchored != null) {
    const [, word, sign, spanText] = anchored;
    const key = word.toLowerCase();
    const anchor: keyof Anchors = key === "t" ? "parent" : (key as keyof Anchors);
    const base = anchors[anchor];
    if (base == null)
      return { ok: false, reason: anchor === "parent" ? "no-parent" : "no-anchor" };
    let offset = TimeSpan.ZERO;
    if (sign != null) {
      const span = parseTimeSpan(spanText);
      if (span == null) return { ok: false, reason: "invalid" };
      offset = sign === "+" ? span.value : span.value.mult(-1);
    }
    return { ok: true, value: base.add(offset), kind: "anchor", anchor, offset };
  }
  const absolute = parseAbsolute(trimmed, anchors.now);
  if (absolute == null) return { ok: false, reason: "invalid" };
  return { ok: true, ...absolute };
};

/**
 * Rounds a form value (nanoseconds as a float64) to the microsecond: a float holds
 * about 0.25 us at today's epoch, so finer digits are noise.
 */
export const roundNumeric = (value: number): bigint => {
  const ns = BigInt(Math.trunc(value));
  // bigint division truncates toward zero, so the half step takes the value's sign.
  return ((ns + (ns < 0n ? -500n : 500n)) / 1000n) * 1000n;
};

/** Converts a form value to an instant, rounded by {@link roundNumeric}. */
export const fromNumeric = (value: number): TimeStamp =>
  new TimeStamp(roundNumeric(value));

/** An instant the way a reading names it: `Today 14:05`. */
export const formatInstant = (value: number, now: TimeStamp): string => {
  const ts = fromNumeric(value);
  return `${TelemText.describeDay(ts, now)} ${TelemText.formatTime(ts)}`;
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

/** Names the unit the caret sits in for a string from `TimeStamp.toPreciseString`. */
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
 * Moves an instant by `steps` of `unit`. Months and years move on the local calendar;
 * every other unit is a fixed span.
 */
export const nudge = (ts: TimeStamp, unit: Unit, steps: number): TimeStamp => {
  const span = UNIT_SPANS[unit];
  if (span != null) return ts.add(span.mult(steps));
  if (unit === "year") return ts.setLocalYear(ts.localYear + steps);
  return ts.setLocalMonth(ts.localMonth + steps);
};

/** The local zone's short name for an instant, e.g. "PDT". */
export const zoneName = (ts: TimeStamp): string => {
  const part = new Intl.DateTimeFormat(undefined, { timeZoneName: "short" })
    .formatToParts(ts.date())
    .find((p) => p.type === "timeZoneName");
  return part?.value ?? "";
};
