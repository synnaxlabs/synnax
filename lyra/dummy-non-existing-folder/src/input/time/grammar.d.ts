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
export declare const parseTimeSpan: (text: string) => ParsedTimeSpan | null;
type EpochUnit = "seconds" | "milliseconds" | "microseconds" | "nanoseconds";
/**
 * How an expression was read. An anchor's `offset` is signed, and zero for the bare
 * word. `clock` is a time of day alone, placed on today; `local` is a local date with
 * or without a time of day.
 */
export type Reading = {
    kind: "anchor";
    anchor: keyof Anchors;
    offset: TimeSpan;
} | {
    kind: "epoch";
    unit: EpochUnit;
} | {
    kind: "iso";
} | {
    kind: "clock";
} | {
    kind: "local";
};
type ParsedTimeStamp = {
    value: TimeStamp;
} & Reading;
/** Why a typed expression did not resolve. */
export type ParseFailure = "invalid" | "no-parent" | "no-anchor";
export type ParseResult = ({
    ok: true;
} & ParsedTimeStamp) | {
    ok: false;
    reason: ParseFailure;
};
/**
 * Parses an instant: an anchor word (`now`, `start`, `end`, `T`) with an optional
 * signed duration, an absolute local date-time (`2026-08-23 14:05:00.250`), a time of
 * day meaning today (`14:05`, `2:05 pm`), an ISO string with a zone, or an epoch in
 * seconds, milliseconds, or nanoseconds.
 */
export declare const parseTimeStamp: (text: string, anchors: Anchors) => ParseResult;
/**
 * Rounds a form value (nanoseconds as a float64) to the microsecond: a float holds
 * about 0.25 us at today's epoch, so finer digits are noise.
 */
export declare const roundNumeric: (value: number) => bigint;
/** Converts a form value to an instant, rounded by {@link roundNumeric}. */
export declare const fromNumeric: (value: number) => TimeStamp;
/**
 * An instant the way a reading names it: `Today 14:05`. Digits below `resolution` are
 * dropped.
 */
export declare const formatInstant: (value: number, now: TimeStamp, resolution?: TimeSpan) => string;
/** A unit of the fixed layout, in caret order. */
export type Unit = "year" | "month" | "day" | "hour" | "minute" | "second" | "millisecond" | "microsecond" | "nanosecond";
/** Names the unit the caret sits in for a string from `TimeStamp.toPreciseString`. */
export declare const unitAt: (caret: number) => Unit;
/**
 * Moves an instant by `steps` of `unit`. Months and years move on the local calendar;
 * every other unit is a fixed span.
 */
export declare const nudge: (ts: TimeStamp, unit: Unit, steps: number) => TimeStamp;
/** The local zone's short name for an instant, e.g. "PDT". */
export declare const zoneName: (ts: TimeStamp) => string;
export {};
//# sourceMappingURL=grammar.d.ts.map