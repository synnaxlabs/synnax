import { bounds, type MultiSeries, type Series, type TimeSpan } from "@synnaxlabs/x";
export declare const seriesOverlap: (x: Series, ys: Series, overlapThreshold: TimeSpan) => boolean;
/**
 * @param xData - the x (timestamp) series.
 * @param yData - the y series, paired with x by time range and alignment overlap.
 * @param xWindow - the x range to bound over.
 * @param overlapThreshold - minimum time range overlap for an x/y pair to count.
 * @param fallback - returned when no paired sample falls inside the window.
 * @returns the bounds of y samples whose paired x value falls inside the window.
 */
export declare const windowBounds: (xData: MultiSeries, yData: MultiSeries, xWindow: bounds.Bounds, overlapThreshold: TimeSpan, fallback: bounds.Bounds) => bounds.Bounds;
//# sourceMappingURL=bounds.d.ts.map