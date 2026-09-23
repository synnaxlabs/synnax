import { table } from "@synnaxlabs/client";
import { type ReactElement } from "react";
export interface RowProps {
    index: number;
    size: number;
    /** Canvas x coordinate of the row's first cell. */
    x: number;
    /** Canvas y coordinate of the row. */
    y: number;
    resourceKey: table.Key;
    cells: string[];
    columns: number[];
    editable: boolean;
    /** True when this is the table's bottom row. */
    last: boolean;
    showIndicator?: boolean;
    onResize: (size: number, index: number) => void;
    onSelect: (index: number, ev: React.MouseEvent) => void;
    onCellSelect: (cellKey: string, ev: React.MouseEvent) => void;
}
export declare const Row: import("react").MemoExoticComponent<({ index, size, x, y, resourceKey, cells, columns, editable, last, showIndicator, onResize, onSelect, onCellSelect, }: RowProps) => ReactElement>;
//# sourceMappingURL=Row.d.ts.map