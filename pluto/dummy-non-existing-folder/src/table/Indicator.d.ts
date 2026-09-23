import "./Table.css";
import { type table } from "@synnaxlabs/client";
import { direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";
/** Pixel size of an indicator strip, mirroring --pluto-table-indicator-size. */
export declare const INDICATOR_SIZE: number;
export declare const getCellColumn: (index: number) => string;
export interface ColumnIndicatorsProps {
    columns: number[];
    rows: table.Row[];
    selected: string[];
    editable: boolean;
    onSelect: (index: number, ev: React.MouseEvent) => void;
    onSelectAll: () => void;
    onResize: (size: number, index: number) => void;
}
export declare const ColumnIndicators: import("react").MemoExoticComponent<({ columns, rows, selected, editable, onSelect, onSelectAll, onResize, }: ColumnIndicatorsProps) => ReactElement>;
export interface IndicatorProps {
    direction: direction.Direction;
    index: number;
    value: number;
    editable: boolean;
    selected?: boolean;
    onChange: (size: number, index: number) => void;
    onSelect: (index: number, ev: React.MouseEvent) => void;
}
export declare const Indicator: ({ direction: dir, index, value, editable, selected, onChange, onSelect, }: IndicatorProps) => ReactElement;
//# sourceMappingURL=Indicator.d.ts.map