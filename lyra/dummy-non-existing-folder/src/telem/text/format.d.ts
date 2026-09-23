import { type TimeSpan, type TimeStamp } from "@synnaxlabs/x";
/**
 * Names the local day of `ts` the way a person would relative to `now`: `Today`,
 * `Yesterday`, `Tomorrow`, a weekday within six days, else `Aug 21`, with the year
 * appended only when it differs from the current one.
 */
export declare const describeDay: (ts: TimeStamp, now: TimeStamp) => string;
/**
 * The local time of day: `14:05`, then `:32` when seconds are set, then the fractional
 * groups of `TimeStamp.toPreciseString` when they are set. Digits below `resolution`
 * are dropped first.
 */
export declare const formatTime: (ts: TimeStamp, resolution?: TimeSpan) => string;
//# sourceMappingURL=format.d.ts.map