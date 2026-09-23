import { Input } from "@synnaxlabs/lyra/input";
import { type NumericTimeRange, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Stage } from "./stage";
/** One thing a commit would change about a range. */
export type Change = {
    kind: "stage";
    from: Stage;
    to: Stage;
} | {
    kind: "set" | "moved" | "cleared";
    bound: Input.Bound;
    before: number;
    after: number;
};
/**
 * Lists what committing `next` would change about `range` besides the end being
 * edited: the stage when the commit crosses now, then each other end it sets, moves,
 * or clears.
 */
export declare const describeChanges: (range: NumericTimeRange, next: NumericTimeRange, editing?: Input.Bound) => Change[];
export interface TimelineEffectProps {
    changes: Change[];
    /** The finest unit the effect shows. */
    resolution?: TimeSpan;
}
/**
 * Renders {@link describeChanges}: a stage change as its two stages, a cleared end in
 * the warning color with the value it loses, and a set or moved end quietly.
 */
export declare const TimelineEffect: ({ changes, resolution, }: TimelineEffectProps) => ReactElement;
//# sourceMappingURL=TimelineEffect.d.ts.map