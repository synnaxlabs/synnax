import { type ReactElement } from "react";
/** Props for {@link ClipboardItems}. */
export interface ClipboardItemsProps {
    cut: () => void;
    copy: () => void;
    paste: () => void;
    hasSelection: boolean;
}
/**
 * Renders the cut, copy, and paste entries of a diagram context menu, wired to the
 * triggers from the diagram's useClipboard. Render inside a {@link Menu}.
 */
export declare const ClipboardItems: ({ cut, copy, paste, hasSelection, }: ClipboardItemsProps) => ReactElement;
//# sourceMappingURL=ClipboardItems.d.ts.map