import { type ReactElement } from "react";
import { Button } from "../button";
/** Props for {@link CopyItem}. */
export interface CopyItemProps extends Button.CopyProps {
    /** The key given to the enclosing menu's `onChange`. */
    itemKey: string;
}
/**
 * A menu entry that copies text to the clipboard and swaps its icon to a check. It
 * takes its level, gap, and background from the enclosing {@link Menu}.
 */
export declare const CopyItem: ({ className, itemKey, ...rest }: CopyItemProps) => ReactElement;
//# sourceMappingURL=CopyItem.d.ts.map