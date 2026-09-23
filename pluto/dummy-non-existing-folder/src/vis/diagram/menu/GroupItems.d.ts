import { type ReactElement } from "react";
/** Props for {@link GroupItems}. */
export interface GroupItemsProps {
    group: () => void;
    ungroup: () => void;
    canGroup: boolean;
    canUngroup: boolean;
}
/**
 * Renders the group and ungroup entries of a diagram context menu, wired to the
 * triggers from the diagram's useTriggers. An entry hides when it does not apply.
 * Render inside a {@link Menu}.
 */
export declare const GroupItems: ({ group, ungroup, canGroup, canUngroup, }: GroupItemsProps) => ReactElement;
//# sourceMappingURL=GroupItems.d.ts.map