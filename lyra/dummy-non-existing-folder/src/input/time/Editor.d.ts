import "./Time.css";
import { type ReactElement, type ReactNode } from "react";
import { type Component } from "../../component";
import { Dialog } from "../../dialog";
import { type Icon } from "../../icon";
import { type Suggestion } from "./suggest";
import { type Variant } from "../types";
/** The props a time input forwards to its trigger, minus the ones it owns. */
export interface BaseProps extends Omit<Dialog.TriggerProps, "value" | "onChange" | "children" | "variant" | "hideCaret"> {
    variant?: Variant;
}
/** A fixed value the editor offers under the readings. */
export interface Action<V> {
    key: string;
    icon: Icon.ReactElement;
    label: string;
    /** An expression that types the same value, shown when it differs from the label. */
    hint?: string;
    /** Called on render and again on a click, so a value like now stays current. */
    value: () => V;
}
export interface EditorProps<V> extends BaseProps {
    /** The value as content, shown at rest. */
    label: ReactNode;
    /** The text the editor opens on. */
    initialText: string;
    /** Lists the ways the text can be read, most likely first. */
    suggest: (text: string) => Suggestion<V>[];
    /** Called with the reading a commit takes. */
    onCommit: (value: V) => void;
    /** Called when a commit finds the field blank. */
    onClear?: () => void;
    /**
     * Moves the unit under the caret by `steps`.
     * @returns the new text, or null when the text has no units; the arrow keys then
     * walk the readings.
     */
    nudge?: (text: string, caret: number, steps: number) => string | null;
    /** Shown while the field is blank. */
    hint: string;
    /** Shown when the text has no reading. */
    unreadMessage: string;
    fieldPlaceholder?: string;
    actions?: Action<V>[];
    /**
     * Rendered under the options with the value the highlighted reading or the hovered
     * action would commit. Use it to say what else a commit would change, and return
     * null when nothing else would.
     */
    effect?: Component.RenderProp<{
        candidate: V;
    }>;
    /** Renders one reading on one line. */
    children: Component.RenderProp<Suggestion<V>>;
}
/**
 * A value that reads as content and edits in a connected dialog under it, the shape
 * of a select. The dialog holds a text field over the readings of its text. Enter or a
 * click outside commits the highlighted reading; Escape discards the edit; Enter on
 * text with no reading keeps the editor open. Hovering a reading highlights it.
 */
export declare const Editor: <V>({ label, initialText, suggest, onCommit, onClear, nudge, hint, unreadMessage, fieldPlaceholder, actions, effect, children, variant, className, style, tooltip, ...rest }: EditorProps<V>) => ReactElement;
//# sourceMappingURL=Editor.d.ts.map