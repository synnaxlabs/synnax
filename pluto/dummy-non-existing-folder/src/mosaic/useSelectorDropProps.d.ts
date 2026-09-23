import { type Tabs } from "@synnaxlabs/lyra/tabs";
export interface UseSelectorDropPropsParams {
    /** The key of the leaf whose tab strip is being wired, as passed to its Leaf. */
    nodeKey: number;
    /**
     * The keys of the tabs currently in the leaf. Used to reject drops that would
     * remove every tab from the leaf and re-insert into it, such as dropping a leaf's
     * sole tab back onto its own strip.
     */
    tabKeys: string[];
}
export type UseSelectorDropPropsReturn = Required<Pick<Tabs.SelectorProps, "haulType" | "canDrop" | "onDrop">>;
/**
 * useSelectorDropProps wires the Tabs.Selector composed inside a {@link Leaf} up
 * as the drop target for the leaf's tab strip. Spread the returned props onto the
 * Selector: strip drops then reach the Frame's onDrop and onCreate handlers with
 * location "center" and the resolved insertion index, claimed before the leaf's
 * own drop target sees them.
 */
export declare const useSelectorDropProps: ({ nodeKey, tabKeys, }: UseSelectorDropPropsParams) => UseSelectorDropPropsReturn;
//# sourceMappingURL=useSelectorDropProps.d.ts.map