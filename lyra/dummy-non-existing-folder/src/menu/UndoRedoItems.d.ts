import { type ReactElement } from "react";
/** Props for {@link UndoRedoItems}. */
export interface UndoRedoItemsProps {
    undo: () => void;
    redo: () => void;
    canUndo: boolean;
    canRedo: boolean;
}
/**
 * Renders the undo and redo entries of a context menu, each hinting the shortcut its
 * host already binds. Render inside a {@link Menu}.
 */
export declare const UndoRedoItems: ({ undo, redo, canUndo, canRedo, }: UndoRedoItemsProps) => ReactElement;
//# sourceMappingURL=UndoRedoItems.d.ts.map