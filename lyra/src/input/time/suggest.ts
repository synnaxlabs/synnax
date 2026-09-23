// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type DurationJSON } from "compromise-dates";

import {
  type Anchors,
  parseTimeSpan,
  parseTimeStamp,
  type Reading,
} from "@/input/time/grammar";

const createLanguage = async () => {
  const [nlp, dates] = await Promise.all([
    import("compromise"),
    import("compromise-dates"),
  ]);
  return nlp.default.extend(dates.default);
};

// The parser is a third of the module's weight and only the phrase fallback needs
// it, so it loads on demand. Until then readings come from the grammar alone.
let language: Awaited<ReturnType<typeof createLanguage>> | null = null;
let loading: Promise<void> | null = null;

/** @returns true once {@link loadLanguage} has resolved. */
export const languageLoaded = (): boolean => language != null;

/** Loads the natural-language parser that reads phrases like `tomorrow at 3pm`. */
export const loadLanguage = (): Promise<void> => {
  loading ??= createLanguage().then((loaded) => {
    language = loaded;
  });
  return loading;
};

/** The end of a range an input holds. Decides what a bare duration or time means. */
export type Bound = "start" | "end";

/** One way the typed text can be read. */
export interface Suggestion<V> {
  key: string;
  value: V;
  /** How the text was read, in the user's words: `5m before now`, `Unix seconds`. */
  reading: string;
}

export interface SuggestOptions {
  anchors: Anchors;
  /** The value the input held when editing began. */
  current?: TimeStamp;
  bound?: Bound;
}

const MAX_SUGGESTIONS = 6;

const ANCHOR_NAMES: Record<keyof Anchors, string> = {
  now: "now",
  start: "the start",
  end: "the end",
  parent: "the parent start",
};

const SIGNED_RE = /^([+-])\s*(.+)$/;

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
  const v = value.date();
  const d = day.date();
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

const stamp = (value: TimeStamp, reading: string): Suggestion<TimeStamp> => ({
  key: value.valueOf().toString(),
  value,
  reading,
});

const describeReading = (reading: Reading): string => {
  switch (reading.kind) {
    case "anchor": {
      const name = ANCHOR_NAMES[reading.anchor];
      const { offset } = reading;
      if (offset.isZero) return name;
      const word = offset.valueOf() > 0n ? "after" : "before";
      return `${spanWords(offset.abs())} ${word} ${name}`;
    }
    case "epoch":
      return `Unix ${reading.unit}`;
    case "iso":
      return "ISO 8601";
    case "clock":
      return "today";
    case "local":
      return "local time";
  }
};

/**
 * Lists the ways the typed text can be read as an instant, most likely first. The
 * grammar's exact reading leads; then a bare time on the other end's day, durations
 * relative to the range, and natural language (`tomorrow at 3pm`, `last monday 8:30`,
 * `5m ago`).
 */
export const suggestTimeStamps = (
  text: string,
  { anchors, current, bound }: SuggestOptions,
): Suggestion<TimeStamp>[] => {
  const out: Suggestion<TimeStamp>[] = [];
  const trimmed = text.trim();
  if (trimmed.length === 0) return [];
  const { now } = anchors;
  const other = bound === "end" ? anchors.start : anchors.end;
  const otherName = bound === "end" ? "the start" : "the end";

  const exact = parseTimeStamp(trimmed, anchors);
  if (exact.ok) out.push(stamp(exact.value, describeReading(exact)));

  if (exact.ok && exact.kind === "clock") {
    if (other != null && !other.isSameDay(now, "local"))
      out.push(stamp(onDayOf(exact.value, other), `on the day of ${otherName}`));
    if (current != null && !current.isSameDay(now, "local"))
      out.push(stamp(onDayOf(exact.value, current), "on the current day"));
  }

  const signed = SIGNED_RE.exec(trimmed);
  if (signed != null) {
    const [, sign, rest] = signed;
    const span = parseTimeSpan(rest)?.value;
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

  const typedSpan = parseTimeSpan(trimmed);
  const unitRun = typedSpan?.kind === "units" ? typedSpan.value : null;
  if (unitRun != null) {
    if (bound === "end" && other != null)
      out.push(stamp(other.add(unitRun), `${spanWords(unitRun)} after the start`));
    if (bound === "start" && other != null)
      out.push(stamp(other.sub(unitRun), `${spanWords(unitRun)} before the end`));
    out.push(stamp(now.add(unitRun), `${spanWords(unitRun)} from now`));
    out.push(stamp(now.sub(unitRun), `${spanWords(unitRun)} ago`));
  }

  // The grammar's exact read is authoritative; language only covers what it misses.
  if (exact.ok || language == null) return distinct(out);

  const phrase = expandAbbreviations(trimmed).toLowerCase();
  const doc = language(phrase);
  const ctx = { today: now.date(), timezone: localZone() };
  const dates = doc.dates(ctx).get();
  for (const d of dates) {
    if (d.start == null) continue;
    const start = new TimeStamp(new Date(d.start));
    const wholeDay = d.unit != null && d.unit !== "time";
    out.push(stamp(start, wholeDay ? `start of ${phrase}` : phrase));
    if (wholeDay && bound === "end" && d.end != null)
      out.push(stamp(new TimeStamp(new Date(d.end)), `end of ${phrase}`));
  }
  if (dates.length === 0 && unitRun == null)
    for (const d of doc.durations().get()) {
      const span = durationToSpan(d);
      if (span.isZero) continue;
      if (bound === "end" && other != null)
        out.push(stamp(other.add(span), `${spanWords(span)} after the start`));
      if (bound === "start" && other != null)
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
export const suggestTimeSpans = (text: string): Suggestion<TimeSpan>[] => {
  const out: Suggestion<TimeSpan>[] = [];
  const trimmed = text.trim();
  if (trimmed.length === 0) return [];
  const exact = parseTimeSpan(trimmed);
  if (exact != null) {
    const reading = exact.kind === "seconds" ? "seconds" : "duration";
    out.push({ key: exact.value.valueOf().toString(), value: exact.value, reading });
  }
  const phrase = normalizeDurationWords(expandAbbreviations(trimmed));
  const normalized = parseTimeSpan(phrase.replace(/\s*(and|,)\s*/g, " "))?.value;
  if (normalized != null)
    out.push({
      key: normalized.valueOf().toString(),
      value: normalized,
      reading: phrase,
    });
  if (language == null) return distinct(out);
  for (const d of language(phrase).durations().get()) {
    const span = durationToSpan(d);
    if (span.isZero) continue;
    out.push({ key: span.valueOf().toString(), value: span, reading: phrase });
  }
  return distinct(out);
};
