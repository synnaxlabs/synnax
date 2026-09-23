import "./Table.css";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ComponentPropsWithRef, type ReactElement, type ReactNode } from "react";
import { type z } from "zod";
import { table as aetherTable } from "./aether";
export { getCellColumn } from "./Indicator";
export interface TableProps extends Omit<ComponentPropsWithRef<"div">, "onCopy" | "onPaste" | "onContextMenu">, Pick<z.infer<typeof aetherTable.Table.stateZ>, "visible"> {
    selected?: string[];
    onSelectionChange?: (next: string[]) => void;
    editable?: boolean;
    onEditableChange?: (editable: boolean) => void;
    showIndicators?: boolean;
    onShowIndicatorsChange?: (next: boolean) => void;
    /** Centers the table in its container. No effect on an axis it overflows. */
    centered?: boolean;
    /** When defined, surfaces a Center / Align item in the context menu. */
    onCenteredChange?: (next: boolean) => void;
    extraMenuItems?: ReactNode;
    enableTriggers?: Triggers.Condition;
}
export declare const Table: ({ selected, onSelectionChange, editable, onEditableChange, showIndicators, onShowIndicatorsChange, centered, onCenteredChange, extraMenuItems, enableTriggers, visible, className, ...rest }: TableProps) => ReactElement;
//# sourceMappingURL=Table.d.ts.map