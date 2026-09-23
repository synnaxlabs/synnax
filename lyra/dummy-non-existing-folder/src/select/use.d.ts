import { type optional, type record } from "@synnaxlabs/x";
import { type UseHoverProps, type UseHoverReturn } from "./useHover";
/**
 * Extra information passed as an additional argument to the `onChange` callback.
 * of the {@link useMultiple} hook.
 */
export interface UseOnChangeExtra<K extends record.Key = record.Key> {
    /** The index of the clicked entry in the list data. */
    clickedIndex: number | null;
    /** The key of the entry that was last clicked. */
    clicked: K | null;
}
/** Props for {@link useSingle} when clicking the selected entry can clear it. */
export interface UseSingleAllowNoneProps<K extends record.Key> {
    value?: K;
    onChange: (next: K | null, extra: UseOnChangeExtra<K>) => void;
    allowNone?: true;
    /** Whether to close the enclosing dialog after a selection. */
    closeDialogOnSelect?: boolean;
    /** Whether to select the first entry whenever the value names nothing in the data. */
    autoSelectOnNone?: boolean;
}
/** Props for {@link useSingle} when a selection is mandatory. */
export interface UseSingleRequiredProps<K extends record.Key> {
    value: K;
    onChange: (next: K, extra: UseOnChangeExtra<K>) => void;
    allowNone: false | undefined;
    closeDialogOnSelect?: boolean;
    autoSelectOnNone?: boolean;
}
type UseSingleInternalProps<K extends record.Key> = UseSingleAllowNoneProps<K> | UseSingleRequiredProps<K>;
/** Props for {@link useSingle}. */
export type UseSingleProps<K extends record.Key> = optional.Optional<UseSingleInternalProps<K>, "allowNone"> & Pick<UseHoverProps<K>, "initialHover" | "enableTriggers">;
/** Props for {@link useMultiple}. */
export interface UseMultipleProps<K extends record.Key> extends Pick<UseHoverProps<K>, "initialHover" | "enableTriggers"> {
    /** Whether the user can deselect the last remaining entry. Defaults to true. */
    allowNone?: boolean;
    value: K[];
    onChange: (next: K[], extra: UseOnChangeExtra<K>) => void;
    /**
     * Whether an unmodified click replaces the selection instead of adding to it. Shift
     * and control still extend and toggle.
     */
    replaceOnSingle?: boolean;
    /** Whether to close the enclosing dialog after a selection. */
    closeDialogOnSelect?: boolean;
    /** Whether to select the first entry whenever the value names nothing in the data. */
    autoSelectOnNone?: boolean;
}
/**
 * hasModifier reports whether a pointer event carries a selection modifier, meaning
 * the gesture is aimed at the selection rather than at the item: shift extends a
 * range from the anchor, control (or command) toggles a single key. A row that
 * activates on click routes to the enclosing selection instead when this is true.
 */
export declare const hasModifier: (e: {
    shiftKey: boolean;
    ctrlKey: boolean;
    metaKey: boolean;
}) => boolean;
/** Return value for the {@link useSingle} and {@link useMultiple} hooks. */
export interface UseReturn<K extends record.Key> extends UseHoverReturn<K> {
    /** Applies a click on the given key, honoring the held modifier keys. */
    onSelect: (key: K) => void;
    /** Replaces the selection outright, ignoring modifiers. */
    setSelected: (keys: K[]) => void;
    /** Empties the selection. A no-op when none is not allowed. */
    clear: () => void;
}
/**
 * Drives a single-entry selection over the enclosing {@link List.Frame}, adding
 * keyboard hover and, when allowed, clear-on-reclick.
 */
export declare const useSingle: <K extends record.Key>({ allowNone, onChange, value, closeDialogOnSelect, initialHover, enableTriggers, autoSelectOnNone, }: UseSingleProps<K>) => UseReturn<K>;
/**
 * Drives a multi-entry selection over the enclosing {@link List.Frame}. Shift extends a
 * range from the last click, control toggles one key, and a plain click adds or removes
 * unless `replaceOnSingle` is set.
 */
export declare const useMultiple: <K extends record.Key>({ value, replaceOnSingle, onChange, initialHover, enableTriggers, allowNone, closeDialogOnSelect, autoSelectOnNone, }: UseMultipleProps<K>) => UseReturn<K>;
export {};
//# sourceMappingURL=use.d.ts.map