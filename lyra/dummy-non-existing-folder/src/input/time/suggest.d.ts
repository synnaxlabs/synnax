import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type Anchors } from "./grammar";
/** @returns true once {@link loadLanguage} has resolved. */
export declare const languageLoaded: () => boolean;
/** Loads the natural-language parser that reads phrases like `tomorrow at 3pm`. */
export declare const loadLanguage: () => Promise<void>;
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
/**
 * Lists the ways the typed text can be read as an instant, most likely first. The
 * grammar's exact reading leads; then a bare time on the other end's day, durations
 * relative to the range, and natural language (`tomorrow at 3pm`, `last monday 8:30`,
 * `5m ago`).
 */
export declare const suggestTimeStamps: (text: string, { anchors, current, bound }: SuggestOptions) => Suggestion<TimeStamp>[];
/**
 * Lists the ways the typed text can be read as a duration: the grammar's exact
 * reading (`2h 30m`, `1:30:00`, `90`), then natural language (`an hour and a half`,
 * `2 hours and 30 minutes`).
 */
export declare const suggestTimeSpans: (text: string) => Suggestion<TimeSpan>[];
//# sourceMappingURL=suggest.d.ts.map