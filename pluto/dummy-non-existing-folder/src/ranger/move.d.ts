import { type NumericTimeRange } from "@synnaxlabs/x";
/** An open start or end. */
export declare const UNSET: number;
/**
 * Commits a start edit. A start moved past the end drags the end with it, keeping
 * the duration. Unscheduling clears the end too.
 */
export declare const moveStart: ({ start, end }: NumericTimeRange, next: number) => NumericTimeRange;
/**
 * Commits an end edit. An end moved before the start drags the start with it,
 * keeping the duration; an open range has none to keep, so the start clamps.
 */
export declare const moveEnd: ({ start, end }: NumericTimeRange, next: number) => NumericTimeRange;
//# sourceMappingURL=move.d.ts.map