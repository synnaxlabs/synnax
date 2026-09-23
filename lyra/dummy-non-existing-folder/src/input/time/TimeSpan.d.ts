import "./Time.css";
import { TimeSpan as XTimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type BaseProps } from "./Editor";
import { type Control } from "../types";
export interface TimeSpanProps extends Control<number>, BaseProps {
    /**
     * When set, the input shows the time elapsed since this instant, ticking every
     * second, instead of `value`. Typing a duration still commits through `onChange`.
     */
    elapsedSince?: number;
    /** The finest unit the label shows; the tooltip and editor keep every digit. */
    resolution?: XTimeSpan;
}
/** Formats a span for the label; zero reads as `0s`. */
export declare const formatTimeSpan: (span: XTimeSpan) => string;
/**
 * A duration input. The value is nanoseconds. At rest it reads `2h 30m 12s`; clicking
 * opens an editor that accepts any duration the grammar does (`30s`, `1:30:00`,
 * `1.5h`) and commits on Enter or a click outside. Unlisted props go to the trigger.
 */
export declare const TimeSpan: ({ value, onChange, elapsedSince, resolution, className, tooltip, ...rest }: TimeSpanProps) => ReactElement;
//# sourceMappingURL=TimeSpan.d.ts.map