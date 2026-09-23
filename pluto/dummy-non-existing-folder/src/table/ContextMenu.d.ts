import { type table } from "@synnaxlabs/client";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement, type ReactNode } from "react";
/** Erases the selected cells. Registered by {@link Table}; shown on the erase item. */
export declare const ERASE_TRIGGER: Triggers.Trigger;
export interface DefaultContextMenuProps {
    resourceKey: table.Key;
    targetID: string | null;
    selected: string[];
    editable: boolean;
    onEditableChange?: (editable: boolean) => void;
    showIndicators?: boolean;
    onShowIndicatorsChange?: (next: boolean) => void;
    /** Whether the table is centered in its container. */
    centered?: boolean;
    /** When defined, surfaces a Center / Align item in the menu. */
    onCenteredChange?: (next: boolean) => void;
    onAddRow: (index?: number) => void;
    onAddCol: (index?: number) => void;
    onRemoveRow: (indices: number[]) => void;
    onRemoveCol: (indices: number[]) => void;
    onEraseCells: (cells: string[]) => void;
    extra?: ReactNode;
}
export declare const DefaultContextMenu: ({ resourceKey, targetID, selected, editable, onEditableChange, showIndicators, onShowIndicatorsChange, centered, onCenteredChange, onAddRow, onAddCol, onRemoveRow, onRemoveCol, onEraseCells, extra, }: DefaultContextMenuProps) => ReactElement;
//# sourceMappingURL=ContextMenu.d.ts.map