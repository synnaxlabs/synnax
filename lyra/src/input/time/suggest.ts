// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import type { DurationJSON } from "compromise-dates";

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

/** The natural-language parser that reads phrases like `tomorrow at 3pm`. */
export type Language = Awaited<ReturnType<typeof createLanguage>>;

// The parser is a third of the module's weight and only the phrase fallback needs
// it, so it loads on demand. Until then readings come from the grammar alone.
let loaded: Language | null = null;
let loading: Promise<Language> | null = null;

/** @returns the parser once {@link loadLanguage} has resolved, or null. */
export const loadedLanguage = (): Language | null => loaded;

/** Loads the natural-language parser, once. */
export const loadLanguage = (): Promise<Language> => {
  loading ??= createLanguage()
    .then((language) => {
      loaded = language;
      return language;
    })
    .catch((err: unknown) => {
      loading = null;
      throw errors.fromUnknown(err);
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
  /** Reads phrases the grammar cannot. Null limits readings to the grammar. */
  language: Language | null;
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

// Looked up in exact case before lowercase, so `Ms` stays apart from `ms`.
const ABBREVIATIONS: Record<string, string> = {
  Ms: "megaseconds",
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
    (_, n: string, unit: string) =>
      `${n} ${ABBREVIATIONS[unit] ?? ABBREVIATIONS[unit.toLowerCase()]}`,
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
  { anchors, language, current, bound }: SuggestOptions,
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

const UNIT_NAMES = new Map<bigint, string>([
  [TimeSpan.NANOSECOND.valueOf(), "nanoseconds"],
  [TimeSpan.MICROSECOND.valueOf(), "microseconds"],
  [TimeSpan.MILLISECOND.valueOf(), "milliseconds"],
  [TimeSpan.SECOND.valueOf(), "seconds"],
  [TimeSpan.MINUTE.valueOf(), "minutes"],
  [TimeSpan.HOUR.valueOf(), "hours"],
  [TimeSpan.DAY.valueOf(), "days"],
]);

const unitName = (unit: TimeSpan): string => {
  const name = UNIT_NAMES.get(unit.valueOf());
  if (name == null) throw new Error(`${unit.toString()} is not a unit of time`);
  return name;
};

/**
 * Lists the ways the typed text can be read as a duration: the grammar's exact
 * reading (`2h 30m`, `1:30:00`, `90`), then natural language (`an hour and a half`,
 * `2 hours and 30 minutes`).
 * @param unit - the unit of a bare number. Defaults to seconds.
 */
export const suggestTimeSpans = (
  text: string,
  language: Language | null,
  unit: TimeSpan = TimeSpan.SECOND,
): Suggestion<TimeSpan>[] => {
  const out: Suggestion<TimeSpan>[] = [];
  const trimmed = text.trim();
  if (trimmed.length === 0) return [];
  const exact = parseTimeSpan(trimmed, unit);
  if (exact != null) {
    const reading = exact.kind === "number" ? unitName(unit) : "duration";
    out.push({ key: exact.value.valueOf().toString(), value: exact.value, reading });
  }
  const phrase = normalizeDurationWords(expandAbbreviations(trimmed));
  const normalized = parseTimeSpan(phrase.replace(/\s*(and|,)\s*/g, " "), unit)?.value;
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

const LAST_MILLISECOND = TimeSpan.milliseconds(999).valueOf();

/**
 * Reads a phrase as the span it names: all of a calendar unit (`yesterday`, `last
 * week`), the time between two times (`1pm to 2pm`), or nothing for an instant (`2
 * hours ago`, `monday 3pm`).
 */
const readPhrase = (
  text: string,
  language: Language,
  now: TimeStamp,
): TimeRange | null => {
  const doc = language(expandAbbreviations(text).toLowerCase());
  const dates = doc.dates({ today: now.date(), timezone: localZone() });
  // Words outside the phrase, such as `test` in `monday test`, name a range instead.
  if (dates.text() !== doc.text()) return null;
  const found = dates.get();
  if (found.length !== 1) return null;
  const [{ start, end, unit }] = found;
  if (start == null) return null;
  const from = new TimeStamp(new Date(start));
  const to = end == null ? from : new TimeStamp(new Date(end));
  // A day, week, or month ends on its last millisecond, and a range excludes its end. A
  // time of day gets the last millisecond of its day too, which is not a real end.
  if (to.valueOf() % TimeSpan.SECOND.valueOf() !== LAST_MILLISECOND)
    return new TimeRange(from, to);
  if (unit == null || unit === "time") return new TimeRange(from, from);
  return new TimeRange(from, to.add(TimeSpan.MILLISECOND));
};

const TIME_OF_DAY_RE =
  /^(?:\d{1,2}:\d{2}\S*(?:\s*[ap]m)?|\d{1,2}\s*[ap]m|noon|midnight)$/i;

interface Side {
  range: TimeRange;
  /** The side names a time of day and no day, as `2pm` does. */
  timeOfDay: boolean;
}

// The same time of day on another day. A day with a clock change is not 24 hours long.
const addDays = (value: TimeStamp, days: number): TimeStamp => {
  const d = value.date();
  const day = new Date(d.getFullYear(), d.getMonth(), d.getDate() + days);
  return onDayOf(value, new TimeStamp(day));
};

/** Reads one side of a range: an instant, or a calendar unit with its whole span. */
const readSide = (text: string, language: Language, now: TimeStamp): Side | null => {
  const trimmed = text.trim();
  const exact = parseTimeStamp(trimmed, { now });
  const range = exact.ok
    ? new TimeRange(exact.value, exact.value)
    : readPhrase(trimmed, language, now);
  if (range == null) return null;
  const timeOfDay = TIME_OF_DAY_RE.test(trimmed);
  // A time of day alone is the last one before now, so `10pm` at 01:00 is last night.
  if (!timeOfDay || range.start.beforeEq(now)) return { range, timeOfDay };
  const { start, end } = range;
  return { range: new TimeRange(addDays(start, -1), addDays(end, -1)), timeOfDay };
};

type RangeForm = (text: string, language: Language, now: TimeStamp) => TimeRange | null;

const FOR_RE = /^(.+?)\s+for\s+(.+)$/i;

/** `2h ago for 1h`, `yesterday 1pm for 30m`. */
const readFor: RangeForm = (text, language, now) => {
  const match = FOR_RE.exec(text);
  if (match == null) return null;
  const from = readSide(match[1], language, now);
  const [length] = suggestTimeSpans(match[2], language);
  if (from == null || length == null) return null;
  const { start } = from.range;
  return new TimeRange(start, start.add(length.value));
};

const SINCE_RE = /^since\s+(.+)$/i;

/** `since 8am`, `since yesterday`. */
const readSince: RangeForm = (text, language, now) => {
  const match = SINCE_RE.exec(text);
  if (match == null) return null;
  const from = readSide(match[1], language, now);
  return from == null ? null : new TimeRange(from.range.start, now);
};

const BETWEEN_RE =
  /^(?:from\s+)?(.+?)(?:\s+(?:to|until|till|through)\s+|\s*→\s*|\s+-\s+)(.+)$/i;

/** `3h ago to 1h ago`, `monday 3pm to tuesday 4pm`, `13:00 to 14:00 yesterday`. */
const readBetween: RangeForm = (text, language, now) => {
  const match = BETWEEN_RE.exec(text);
  if (match == null) return null;
  const from = readSide(match[1], language, now);
  const to = readSide(match[2], language, now);
  if (from == null || to == null) return null;
  let start = from.range.start;
  let end = to.range.end;
  // A time of day alone falls on the day of the other side.
  if (to.timeOfDay && !from.timeOfDay) end = onDayOf(end, start);
  if (from.timeOfDay && !to.timeOfDay) start = onDayOf(start, to.range.start);
  // An end before the start crosses midnight: `10pm to 2am` ends the next day, and
  // `10pm to 2am yesterday` starts the day before.
  const crosses = end.beforeEq(start);
  if (crosses && to.timeOfDay) end = addDays(end, 1);
  else if (crosses && from.timeOfDay) start = addDays(start, -1);
  return new TimeRange(start, end);
};

const INSTANT_LENGTH = TimeSpan.HOUR;

/**
 * `yesterday`, `nov 17 9am-10am`: the span the text names. An instant alone (`1h ago`,
 * `yesterday 1pm`) starts an hour, cut short at now.
 */
const readWhole: RangeForm = (text, language, now) => {
  const whole = readSide(text, language, now);
  if (whole == null) return null;
  const { range } = whole;
  if (!range.span.isZero) return range;
  const end = range.start.add(INSTANT_LENGTH);
  return new TimeRange(range.start, end.valueOf() > now.valueOf() ? now : end);
};

const RANGE_FORMS: RangeForm[] = [readFor, readSince, readBetween, readWhole];

/**
 * Reads the text as one time range. It reads a start and a length (`2h ago for 1h`),
 * a start until now (`since 8am`), two ends (`3h ago to 1h ago`, `13:00 to 14:00
 * yesterday`), and phrases (`yesterday from 1pm to 2pm`, `nov 17 9am-10am`). A day,
 * week, or month covers all of it. An instant alone (`1h ago`) starts an hour that
 * ends at now at the latest.
 * @returns the range, or null when the whole text is not one range of positive length.
 */
export const suggestTimeRange = (
  text: string,
  language: Language,
  now: TimeStamp,
): TimeRange | null => {
  const trimmed = text.trim();
  for (const read of RANGE_FORMS) {
    const range = read(trimmed, language, now);
    if (range != null && range.isValid && !range.span.isZero) return range;
  }
  return null;
};
