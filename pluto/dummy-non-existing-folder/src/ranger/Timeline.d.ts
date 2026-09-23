import "./Timeline.css";
import { type Button } from "@synnaxlabs/lyra/button";
import { type Component } from "@synnaxlabs/lyra/component";
import { Input } from "@synnaxlabs/lyra/input";
import { type NumericTimeRange, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface StageButtonProps extends Input.Control<NumericTimeRange> {
    /** Shows the icon alone, for list rows. */
    iconOnly?: boolean;
    variant?: Button.Variant;
    level?: text.Level;
    size?: Component.Size;
    disabled?: boolean;
    preview?: boolean;
    className?: string;
}
/**
 * The range's stage as a chip whose menu holds the transitions out of it. A
 * transition writes the timestamps that define the next stage: Start stamps the
 * start, Complete stamps the end, Reopen clears it. The menu says what the hovered
 * transition would change.
 */
export declare const StageButton: ({ value, onChange, iconOnly, variant, level, size, disabled, preview, className, }: StageButtonProps) => ReactElement;
export interface TimelineProps extends Input.Control<NumericTimeRange> {
    /** The parent range, which enables `T+` expressions and actions. */
    parent?: NumericTimeRange;
    variant?: Input.Variant;
    level?: text.Level;
    size?: Component.Size;
    disabled?: boolean;
    preview?: boolean;
    className?: string;
}
/**
 * A range's stage and the timestamps that define it, in one row. To do shows the
 * planned start and, once it is set, the planned end; In progress shows the start, the
 * elapsed time, and the planned end; Completed shows start, end, and duration.
 * Editing a timestamp keeps the other in place; an edit that crosses it slides it to
 * keep the duration. The editor says what the highlighted reading would move and
 * which stage it would land the range in before it is taken. Editing the duration
 * moves the end. The stage chip's menu holds the transitions, which stamp the
 * timestamps.
 */
export declare const Timeline: ({ value, onChange, parent, variant, level, size, disabled, preview, className, }: TimelineProps) => ReactElement;
//# sourceMappingURL=Timeline.d.ts.map