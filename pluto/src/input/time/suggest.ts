// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import compromise from "compromise";
import compromiseDates, { type DurationJSON } from "compromise-dates";

import {
  type Anchors,
  parseTimeSpan,
  parseTimeStamp,
  sameDay,
  toDate,
} from "@/input/time/grammar";

const nlp = compromise.extend(compromiseDates);

/** The cell's place in a range, which decides what a bare duration or time means. */
export type Role = "start" | "end";

/** One way the typed text can be read. */
export interface Suggestion {
  key: string;
  value: TimeStamp;
  /** How the text was read, in the user's words: `5m before now`, `Unix seconds`. */
  reading: string;
}

export interface SuggestOptions {
  anchors: Anchors;
  /** The value the cell held when editing began. */
  current?: TimeStamp;
  role?: Role;
}

export interface SpanSuggestion {
  key: string;
  value: TimeSpan;
  reading: string;
}

const MAX_SUGGESTIONS = 6;

const ANCHOR_NAMES: Record<keyof Anchors, string> = {
  now: "now",
  start: "the start",
  end: "the end",
  parent: "the parent start",
};

const ANCHORED_RE = /^(now|start|end|t)\s*(?:([+-])\s*(.+))?$/i;
const DIGITS_RE = /^\d{9,19}$/;
const ISO_ZONE_RE = /^\d{4}-\d{2}-\d{2}T.*(Z|[+-]\d{2}:?\d{2})$/;
const CLOCK_ONLY_RE = /^\d{1,2}(?::\d{2}){0,2}(?:\.\d+)?\s*(?:am|pm)?$/i;
const SIGNED_RE = /^([+-])\s*(.+)$/;
const UNIT_RUN_RE =
  /^\d+(?:\.\d+)?\s*(?:ms|µs|us|ns|min|d|h|m|s)(?:\s*\d+(?:\.\d+)?\s*(?:ms|µs|us|ns|min|d|h|m|s))*$/i;

const EPOCH_UNITS: [string, TimeSpan][] = [
  ["seconds", TimeSpan.SECOND],
  ["milliseconds", TimeSpan.MILLISECOND],
  ["microseconds", TimeSpan.MICROSECOND],
  ["nanoseconds", TimeSpan.NANOSECOND],
];

const EPOCH_MIN = new TimeStamp(new Date(1980, 0, 1));
const EPOCH_MAX = new TimeStamp(new Date(2100, 0, 1));

const ABBREVIATIONS: Record<string, string> = {
  ms: "milliseconds",
  s: "seconds",
  sec: "seconds",
  secs: "seconds",
  m: "minutes",
  min: "minutes",
  mins: "minutes",
  h: "hours",
  hr: "hours",
  hrs: "hours",
  d: "days",
  w: "weeks",
  wk: "weeks",
  wks: "weeks",
};

/** Spells out `5m ago` as `5 minutes ago` so the language parser can read it. */
const expandAbbreviations = (text: string): string =>
  text.replace(
    /(\d+(?:\.\d+)?)\s*(ms|secs?|s|mins?|m|hrs?|h|d|wks?|w)\b/gi,
    (_, n: string, unit: string) => `${n} ${ABBREVIATIONS[unit.toLowerCase()]}`,
  );

const localZone = (): string => Intl.DateTimeFormat().resolvedOptions().timeZone;

/** Moves `value` to the calendar day of `day`, keeping its time of day. */
const onDayOf = (value: TimeStamp, day: TimeStamp): TimeStamp => {
  const v = toDate(value);
  const d = toDate(day);
  const moved = new Date(
    d.getFullYear(),
    d.getMonth(),
    d.getDate(),
    v.getHours(),
    v.getMinutes(),
    v.getSeconds(),
    v.getMilliseconds(),
  );
  return new TimeStamp(moved).add(new TimeSpan(value.valueOf() % 1000000n));
};

const spanWords = (span: TimeSpan): string => span.toString("full") || "0s";

const durationToSpan = (d: DurationJSON): TimeSpan =>
  TimeSpan.days((d.year ?? 0) * 365 + (d.month ?? 0) * 30 + (d.week ?? 0) * 7)
    .add(TimeSpan.days(d.day ?? 0))
    .add(TimeSpan.hours(d.hour ?? 0))
    .add(TimeSpan.minutes(d.minute ?? 0))
    .add(TimeSpan.seconds(d.second ?? 0))
    .add(TimeSpan.milliseconds(d.millisecond ?? 0));

/** Keeps the first reading of each distinct value, up to the cap. */
const distinct = <S extends { value: { valueOf(): bigint } }>(items: S[]): S[] => {
  const seen = new Set<bigint>();
  const out: S[] = [];
  for (const item of items) {
    const ns = item.value.valueOf();
    if (seen.has(ns)) continue;
    seen.add(ns);
    out.push(item);
    if (out.length === MAX_SUGGESTIONS) break;
  }
  return out;
};

const stamp = (value: TimeStamp, reading: string): Suggestion => ({
  key: value.valueOf().toString(),
  value,
  reading,
});

/**
 * Lists the ways the typed text can be read as an instant, most likely first. The
 * grammar's exact reading leads; then the alternatives it hides (other epoch units,
 * a bare time on the other end's day), durations relative to the range, and natural
 * language (`tomorrow at 3pm`, `last monday 8:30`, `5m ago`).
 */
export const suggestTimeStamps = (
  text: string,
  { anchors, current, role }: SuggestOptions,
): Suggestion[] => {
  const out: Suggestion[] = [];
  const trimmed = text.trim();
  if (trimmed.length === 0) return [];
  const { now } = anchors;
  const other = role === "end" ? anchors.start : anchors.end;
  const otherName = role === "end" ? "the start" : "the end";

  const exact = parseTimeStamp(trimmed, anchors);
  if (exact.ok) {
    const anchored = ANCHORED_RE.exec(trimmed);
    if (anchored != null) {
      const [, , sign, spanText] = anchored;
      const name = ANCHOR_NAMES[exact.anchor ?? "now"];
      const span = spanText == null ? null : parseTimeSpan(spanText);
      if (span == null || span.isZero) out.push(stamp(exact.value, name));
      else
        out.push(
          stamp(
            exact.value,
            `${spanWords(span)} ${sign === "+" ? "after" : "before"} ${name}`,
          ),
        );
    } else if (DIGITS_RE.test(trimmed)) {
      const unit =
        trimmed.length <= 10
          ? "seconds"
          : trimmed.length <= 13
            ? "milliseconds"
            : "nanoseconds";
      out.push(stamp(exact.value, `Unix ${unit}`));
    } else if (ISO_ZONE_RE.test(trimmed)) out.push(stamp(exact.value, "ISO 8601"));
    else if (CLOCK_ONLY_RE.test(trimmed)) out.push(stamp(exact.value, "today"));
    else out.push(stamp(exact.value, "local time"));
  }

  if (DIGITS_RE.test(trimmed)) {
    const n = BigInt(trimmed);
    for (const [name, unit] of EPOCH_UNITS) {
      const value = new TimeStamp(n * unit.valueOf());
      if (value.before(EPOCH_MIN) || value.after(EPOCH_MAX)) continue;
      out.push(stamp(value, `Unix ${name}`));
    }
  }

  if (exact.ok && CLOCK_ONLY_RE.test(trimmed)) {
    if (other != null && !sameDay(other, now))
      out.push(stamp(onDayOf(exact.value, other), `on the day of ${otherName}`));
    if (current != null && !sameDay(current, now))
      out.push(stamp(onDayOf(exact.value, current), "on the current day"));
  }

  const signed = SIGNED_RE.exec(trimmed);
  if (signed != null) {
    const [, sign, rest] = signed;
    const span = parseTimeSpan(rest);
    if (span != null) {
      const word = sign === "+" ? "after" : "before";
      const shift = (base: TimeStamp): TimeStamp =>
        sign === "+" ? base.add(span) : base.sub(span);
      if (current != null)
        out.push(stamp(shift(current), `${spanWords(span)} ${word} the current value`));
      if (other != null)
        out.push(stamp(shift(other), `${spanWords(span)} ${word} ${otherName}`));
      out.push(stamp(shift(now), `${spanWords(span)} ${word} now`));
    }
  }

  const unitRun = UNIT_RUN_RE.test(trimmed) ? parseTimeSpan(trimmed) : null;
  if (unitRun != null) {
    if (role === "end" && other != null)
      out.push(stamp(other.add(unitRun), `${spanWords(unitRun)} after the start`));
    if (role === "start" && other != null)
      out.push(stamp(other.sub(unitRun), `${spanWords(unitRun)} before the end`));
    out.push(stamp(now.add(unitRun), `${spanWords(unitRun)} from now`));
    out.push(stamp(now.sub(unitRun), `${spanWords(unitRun)} ago`));
  }

  // The grammar's exact read is authoritative; language only covers what it misses.
  if (exact.ok) return distinct(out);

  const phrase = expandAbbreviations(trimmed).toLowerCase();
  const doc = nlp(phrase);
  const ctx = { today: toDate(now), timezone: localZone() };
  const dates = doc.dates(ctx).get();
  for (const d of dates) {
    if (d.start == null) continue;
    const start = new TimeStamp(new Date(d.start));
    const wholeDay = d.unit != null && d.unit !== "time";
    out.push(stamp(start, wholeDay ? `start of ${phrase}` : phrase));
    if (wholeDay && role === "end" && d.end != null)
      out.push(stamp(new TimeStamp(new Date(d.end)), `end of ${phrase}`));
  }
  if (dates.length === 0 && unitRun == null)
    for (const d of doc.durations().get()) {
      const span = durationToSpan(d);
      if (span.isZero) continue;
      if (role === "end" && other != null)
        out.push(stamp(other.add(span), `${spanWords(span)} after the start`));
      if (role === "start" && other != null)
        out.push(stamp(other.sub(span), `${spanWords(span)} before the end`));
      out.push(stamp(now.add(span), `${spanWords(span)} from now`));
    }

  return distinct(out);
};

const WORD_NUMBERS: Record<string, number> = {
  a: 1,
  an: 1,
  one: 1,
  two: 2,
  three: 3,
  four: 4,
  five: 5,
  six: 6,
  seven: 7,
  eight: 8,
  nine: 9,
  ten: 10,
  eleven: 11,
  twelve: 12,
  fifteen: 15,
  twenty: 20,
  thirty: 30,
  forty: 40,
  fifty: 50,
  sixty: 60,
  ninety: 90,
  half: 0.5,
  quarter: 0.25,
};

/** Rewrites `an hour and a half` as `1.5 hours` so the parsers can read it. */
const normalizeDurationWords = (text: string): string => {
  let t = text.toLowerCase().trim();
  t = t.replace(
    /\b(half|quarter) (?:of )?(?:an?|one) (hour|minute|second)\b/g,
    (_, f, u) => `${WORD_NUMBERS[f]} ${u}s`,
  );
  t = t.replace(
    /\b(\w+) (hours?|minutes?|seconds?|days?) and a half\b/g,
    (_, n: string, u: string) => `${(WORD_NUMBERS[n] ?? Number(n)) + 0.5} ${u}`,
  );
  t = t.replace(
    /\b(\w+) and a half (hours?|minutes?|seconds?|days?)\b/g,
    (_, n, u) => `${(WORD_NUMBERS[n] ?? Number(n)) + 0.5} ${u}`,
  );
  t = t.replace(
    /\b(a|an|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|fifteen|twenty|thirty|forty|fifty|sixty|ninety)\b/g,
    (w: string) => String(WORD_NUMBERS[w]),
  );
  return t;
};

/**
 * Lists the ways the typed text can be read as a duration: the grammar's exact
 * reading (`2h 30m`, `1:30:00`, `90`), then natural language (`an hour and a half`,
 * `2 hours and 30 minutes`).
 */
export const suggestTimeSpans = (text: string): SpanSuggestion[] => {
  const out: SpanSuggestion[] = [];
  const trimmed = text.trim();
  if (trimmed.length === 0) return [];
  const exact = parseTimeSpan(trimmed);
  if (exact != null) {
    const reading = /^\d+(?:\.\d+)?$/.test(trimmed) ? "seconds" : "duration";
    out.push({ key: exact.valueOf().toString(), value: exact, reading });
  }
  const phrase = normalizeDurationWords(expandAbbreviations(trimmed));
  const normalized = parseTimeSpan(phrase.replace(/\s*(and|,)\s*/g, " "));
  if (normalized != null)
    out.push({
      key: normalized.valueOf().toString(),
      value: normalized,
      reading: phrase,
    });
  for (const d of nlp(phrase).durations().get()) {
    const span = durationToSpan(d);
    if (span.isZero) continue;
    out.push({ key: span.valueOf().toString(), value: span, reading: phrase });
  }
  return distinct(out);
};
