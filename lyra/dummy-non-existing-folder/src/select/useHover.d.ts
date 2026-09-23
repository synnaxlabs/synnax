import { type record } from "@synnaxlabs/x";
import { Triggers } from "../triggers";
/** Props for {@link useHover}. */
export interface UseHoverProps<K extends record.Key> {
    /** Index hovered when the dialog opens. Defaults to none. */
    initialHover?: number;
    data: K[];
    onSelect: (key: K) => void;
    /**
     * When to answer keyboard triggers. Defaults to the enclosing dialog's visibility.
     */
    enableTriggers?: Triggers.Condition;
}
/** Return value for {@link useHover}. */
export interface UseHoverReturn<K extends record.Key> {
    /** The key the arrow keys currently rest on. */
    hover: K;
}
/**
 * Moves a hover cursor through the list with the arrow keys and selects with Enter,
 * scrolling the hovered item into view. Holding an arrow key repeats.
 */
export declare const useHover: <K extends record.Key>({ data, initialHover, onSelect, enableTriggers, }: UseHoverProps<K>) => UseHoverReturn<K>;
//# sourceMappingURL=useHover.d.ts.map