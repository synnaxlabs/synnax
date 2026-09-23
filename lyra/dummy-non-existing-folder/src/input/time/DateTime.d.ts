import "./Time.css";
import { type NumericTimeRange, type TimeSpan as XTimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Component } from "../../component";
import { type BaseProps } from "./Editor";
import { type Bound } from "./suggest";
import { type Control } from "../types";
/** The instants an input may anchor typed expressions on, as form values. */
export interface DateTimeAnchors {
    /** The other end of the range the input belongs to. */
    start?: number;
    end?: number;
    /** The parent range, the `T` in `T+3.2s`. */
    parent?: NumericTimeRange;
}
export interface DateTimeProps extends Control<number>, BaseProps {
    anchors?: DateTimeAnchors;
    /**
     * An instant whose day is already shown beside this input; the label drops its own
     * day when the two match.
     */
    sharedDay?: number;
    /** The end of a range the input holds; decides what a bare duration or time means. */
    bound?: Bound;
    /**
     * Rendered under the options with the value the highlighted reading or the hovered
     * action would commit. Use it to say what else a commit would change, and return
     * null when nothing else would.
     */
    effect?: Component.RenderProp<{
        candidate: number;
    }>;
    /** The finest unit the label shows; the tooltip and editor keep every digit. */
    resolution?: XTimeSpan;
    /**
     * Renders a value equal to this as an empty input and commits it when the field is
     * cleared.
     */
    emptyValue?: number;
    /** What the empty input says at rest, as a prompt: `Set a start time`. */
    placeholder?: string;
    /** The action that clears the input back to `emptyValue`. */
    clearLabel?: string;
}
/**
 * A date-time input. The value is nanoseconds since the Unix epoch. At rest it reads
 * as content (`Today 14:05:32`); clicking opens an editor with the instant in a fixed
 * local layout, where typing replaces it with an expression and Up and Down nudge
 * the unit under the caret (Shift for ten). Enter or a click outside commits. Unlisted
 * props go to the trigger.
 */
export declare const DateTime: ({ value, onChange, anchors: { start, end, parent }, emptyValue, placeholder, sharedDay, bound, effect, resolution, clearLabel, className, tooltip, ...rest }: DateTimeProps) => ReactElement;
//# sourceMappingURL=DateTime.d.ts.map