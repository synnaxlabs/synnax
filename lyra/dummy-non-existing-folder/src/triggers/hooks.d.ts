import { type xy } from "@synnaxlabs/x";
import { type RefObject } from "react";
import { type Condition } from "./Scope";
import { type MatchOptions, type Stage, type Trigger } from "./triggers";
/** The event {@link use} hands its callback. */
export interface UseEvent {
    target: HTMLElement;
    prevTriggers: Trigger[];
    /** The matched triggers this stage applies to. */
    triggers: Trigger[];
    stage: Stage;
    cursor: xy.XY;
    /**
     * Prevents the event from being dispatched to any remaining Triggers.use
     * subscribers with a lower priority than the current one. Subscribers at the
     * same priority that have not yet been notified still receive the event.
     */
    stopPropagation: () => void;
}
/** Props for {@link use}. */
export interface UseProps extends MatchOptions {
    triggers?: Trigger | Trigger[];
    /** Fires only while the cursor sits inside this element. */
    region?: RefObject<HTMLElement | null>;
    callback?: (e: UseEvent) => void;
    /** Whether the event target must be the region itself, not a descendant. */
    regionMustBeElement?: boolean;
    /**
     * Withholds events from this subscriber while it resolves false. Use it for conditions
     * the subscriber itself owns, such as whether its content is editable. Whether the
     * surrounding view is the one the user is working in belongs in a {@link Scope}.
     */
    enabled?: Condition;
    /**
     * Priority of this subscriber. Higher-priority subscribers receive events
     * before lower-priority ones and may call stopPropagation on the event to
     * prevent lower-priority subscribers from receiving it. Defaults to 0.
     */
    priority?: number;
}
/**
 * Binds a keyboard or mouse shortcut for as long as the caller is mounted. The callback
 * fires on press and again on release, and only while the enclosing {@link Scope} is
 * the active one.
 *
 * @example
 * Triggers.use({ triggers: [["Control", "S"]], callback: ({ stage }) => save(stage) });
 */
export declare const use: ({ triggers, callback: f, region, loose, double, regionMustBeElement, priority, enabled, }: UseProps) => void;
/** Props for {@link useUndoRedo}. */
export interface UseUndoRedoProps {
    undo: () => void;
    redo: () => void;
    enabled?: Condition;
}
/** useUndoRedo binds the standard undo and redo shortcuts to the given handlers. */
export declare const useUndoRedo: ({ undo, redo, enabled }: UseUndoRedoProps) => void;
/** Which of the watched triggers are down right now. */
export interface UseHeldReturn {
    triggers: Trigger[];
    held: boolean;
}
/** Props for {@link useHeld} and {@link useHeldRef}. */
export interface UseHeldProps {
    triggers: Trigger[];
    /** Whether a superset of a trigger still counts as held. */
    loose?: boolean;
}
/**
 * Tracks which of the given triggers are down, in a ref rather than state. Use it in an
 * event handler that reads the modifier keys, where a re-render per keypress is waste.
 */
export declare const useHeldRef: ({ triggers, loose, }: UseHeldProps) => RefObject<UseHeldReturn>;
/**
 * Tracks which of the given triggers are down and re-renders the caller on every
 * change. Prefer {@link useHeldRef} when only an event handler reads the result.
 */
export declare const useHeld: ({ triggers, loose }: UseHeldProps) => UseHeldReturn;
//# sourceMappingURL=hooks.d.ts.map