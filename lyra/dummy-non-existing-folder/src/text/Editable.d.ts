import "./Editable.css";
import { type ReactElement } from "react";
import { type Input } from "../input";
import { type state } from "../state";
import { type TextProps } from "./Text";
/** Props for {@link Editable}. */
export type EditableProps = Omit<TextProps<"p">, "children" | "onChange"> & Omit<Input.Control<string>, "onChange"> & {
    /** Receives the committed text. Return false to reject it, reverting the field to
     * value; the field otherwise holds the committed text until value catches up. */
    onChange: (value: string) => void | boolean;
    /** Lifts the editing flag out of the component so a parent can drive it. */
    useEditableState?: state.PureUse<boolean>;
    /** Whether a double click starts editing. Defaults to true. */
    allowDoubleClick?: boolean;
    /** Whether an empty value commits. It reverts otherwise. Defaults to false. */
    allowEmpty?: boolean;
    /** Whether to outline the field while editing. Defaults to true. */
    outline?: boolean;
};
/** Options for {@link asyncEdit}. */
export interface AsyncEditOptions {
    /** Replaces the displayed text for the duration of the edit. Use it when the field
     * shows a derived value (e.g. an alias) but the edit targets the underlying one.
     * Committing it unchanged counts as a cancel, and the displayed text comes back. */
    initialValue?: string;
}
/**
 * Starts editing the {@link Editable} carrying the given id, retrying for a second
 * while it mounts. Use it to rename a resource the same gesture just created.
 *
 * @returns the final text and whether the user committed it. A rejected promise means
 * no such element appeared.
 */
export declare const asyncEdit: (id: string, options?: AsyncEditOptions) => Promise<[string, boolean]>;
/** Starts editing the {@link Editable} carrying the given id, discarding the result. */
export declare const edit: (id: string, options?: AsyncEditOptions) => void;
/**
 * Text that turns into a field in place. Commits on Enter or blur, reverts on Escape.
 * Give it an id to drive it from elsewhere with {@link edit}.
 */
export declare const Editable: ({ ref: propsRef, onChange, value, className, useEditableState, allowDoubleClick, onDoubleClick, allowEmpty, style, outline, ...rest }: EditableProps) => ReactElement;
/** Props for {@link MaybeEditable}. */
export type MaybeEditableProps = Omit<EditableProps, "onChange"> & {
    /** A handler makes the text editable. true makes it editable but discards the edit;
     * false or absent renders plain text. */
    onChange?: EditableProps["onChange"] | boolean;
    /** Forces plain text whatever onChange says. */
    disabled?: boolean;
};
/**
 * Renders {@link Editable} when the caller can accept an edit and plain {@link Text}
 * when it cannot, so a caller gated on a permission needs no branch of its own.
 */
export declare const MaybeEditable: ({ onChange, disabled, value, allowDoubleClick, ...rest }: MaybeEditableProps) => ReactElement;
//# sourceMappingURL=Editable.d.ts.map