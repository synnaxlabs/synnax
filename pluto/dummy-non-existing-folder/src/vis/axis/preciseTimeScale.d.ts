import { type CrudeTimeStamp, TimeSpan, TimeStamp } from "@synnaxlabs/x";
/** Tick step sizes, from 1 nanosecond to 1 second in 1-2-5 increments. */
export declare const TIME_SCALE_STEPS: TimeSpan[];
export interface PreciseTimeScaleProps {
    /** The domain of the time scale as [start, end] timestamps */
    domain: [CrudeTimeStamp, CrudeTimeStamp];
    /** The range of the scale as [start, end] numbers for visual representation */
    range: [number, number];
}
/**
 * Maps a time domain onto a numeric range, holding nanosecond precision in BigInt
 * throughout.
 *
 * @example preciseTimeScale().domain([start, end]).range([0, 1000]).scale(t)
 */
export declare class PreciseTimeScale {
    private _domain;
    private _range;
    private _span;
    constructor();
    /** Reads the time domain, or sets it and returns this for chaining. */
    domain(): [TimeStamp, TimeStamp];
    domain(domain: [CrudeTimeStamp, CrudeTimeStamp]): this;
    /** Reads the numeric range, or sets it and returns this for chaining. */
    range(): [number, number];
    range(range: [number, number]): this;
    /** @returns the position of the timestamp within the range. */
    scale(value: CrudeTimeStamp): number;
    /**
     * @returns evenly spaced tick positions across the domain. The count is a target: the
     * step is rounded to a {@link TIME_SCALE_STEPS} interval, so the result may differ.
     */
    ticks(count: number): TimeStamp[];
    /** @returns the smallest {@link TIME_SCALE_STEPS} entry that fits targetCount ticks. */
    private calculateOptimalStep;
    /** Formats a tick label, in microseconds below a 50µs span and milliseconds above. */
    formatTick(value: TimeStamp): string;
}
export declare const preciseTimeScale: () => PreciseTimeScale;
//# sourceMappingURL=preciseTimeScale.d.ts.map