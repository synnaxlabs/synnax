import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { Store } from "../store";
type Value<K extends record.Key = record.Key> = Store.MembershipValue<K>;
interface SelectionState<K extends record.Key = record.Key> {
    value: Value<K>;
    hover?: K;
}
interface ContextValue<K extends record.Key = record.Key> {
    onSelect: (key: K) => void;
    setSelected: (keys: K[]) => void;
    clear: () => void;
    subscribe: (listener: () => void, key?: K) => () => void;
    getState: () => SelectionState<K>;
}
/** Props for {@link Context}. */
export interface ContextProps<K extends record.Key = record.Key> extends PropsWithChildren, Partial<Pick<ContextValue<K>, "onSelect" | "setSelected" | "clear">>, SelectionState<K> {
}
/** Return value for {@link useItemState}. */
export interface UseItemStateReturn {
    selected: boolean;
    /**
     * focused is true when the key heads an ordered multi-selection: the value is an
     * array and this key is its first element. Always false for scalar selections.
     */
    focused: boolean;
    head: boolean;
    hovered: boolean;
    onSelect: () => void;
}
/**
 * Context distributes a controlled selection to keyed item consumers. Membership, focus,
 * and hover are held in independent stores, so an item re-renders only for the dimensions
 * it reads via useItemState. Focus tracks the head of an ordered multi-selection.
 */
export declare const Context: <K extends record.Key = record.Key>({ value, onSelect, setSelected, clear, hover: hoverValue, children, }: ContextProps<K>) => ReactElement;
/** useContext returns the enclosing selection's imperative handle. */
export declare const useContext: <K extends record.Key = record.Key>() => ContextValue<K>;
/**
 * useItemState subscribes a single keyed item to the enclosing Context, re-rendering only
 * when that key's selected, focused, or hovered state flips.
 */
/**
 * Reads one key's place in the enclosing selection. The caller re-renders only when
 * the dimensions it reads change, so a long list stays cheap.
 */
export declare const useItemState: <K extends record.Key>(key: K) => UseItemStateReturn;
/**
 * useSelectedAmong returns the selected key among the given keys, or undefined when none
 * of them is selected. It subscribes only to the given keys, so consumers stay isolated
 * from changes to the rest of the selection. When more than one of the keys is selected,
 * the earliest in the selection's order wins.
 */
export declare const useSelectedAmong: <K extends record.Key = record.Key>(keys: K[]) => K | undefined;
/** useSelected returns the currently selected keys. */
export declare const useSelected: <K extends record.Key = record.Key>() => K[];
/** useClear returns a callback that clears the enclosing selection. */
export declare const useClear: () => (() => void);
export {};
//# sourceMappingURL=Context.d.ts.map