import "./Table.css";
import { type ComponentPropsWithRef, type ReactElement } from "react";
import { type RenderProp } from "../component/renderProp";
import { type Control } from "./types";
/** The value one {@link Table} cell holds. */
export type TableCell = string | number;
/** The props a {@link TableColumn} cell renderer receives. */
export interface TableCellProps<V extends TableCell = TableCell> extends Control<V> {
    /** Whether the cell is read-only. */
    preview?: boolean;
    "aria-label": string;
}
interface BaseTableColumnProps {
    /** The heading rendered above the column. Omitted for a lone unlabeled column. */
    name?: string;
}
/**
 * The column's type decides what a new row holds, how pasted text is read, and which
 * input a cell defaults to.
 */
export type TableColumnProps = (BaseTableColumnProps & {
    type?: "number";
    /** Renders the cell. Defaults to a numeric input. */
    children?: RenderProp<TableCellProps<number>>;
}) | (BaseTableColumnProps & {
    type: "string";
    /** Renders the cell. Defaults to a text input. */
    children?: RenderProp<TableCellProps<string>>;
});
/**
 * Declares one column of a {@link Table}. It renders nothing itself: the table reads
 * its props to build the header and every row's cell.
 *
 * @example <Input.TableColumn name="Pre-scaled" />
 * @example <Input.TableColumn name="Label" type="string" />
 */
export declare const TableColumn: (_: TableColumnProps) => null;
type ColumnElement = ReactElement<TableColumnProps>;
export interface TableProps extends Omit<ComponentPropsWithRef<"table">, "onChange" | "children">, Control<TableCell[][]> {
    /** The {@link TableColumn} children in render order. */
    children: ColumnElement | ColumnElement[];
    /** Hides the add and remove buttons and makes every cell read-only. */
    preview?: boolean;
    /** Labels a row's gutter cell. Defaults to the one-based row index. */
    rowLabel?: (index: number) => string;
    /** Builds the row the add button appends. Defaults to the column default values. */
    createRow?: (value: TableCell[][]) => TableCell[];
}
/**
 * An editable grid. The value is row-major: one entry per row, holding one value per
 * column. Enter and the up and down arrows move between cells. Pasting a tab or comma
 * delimited block into a focused cell fills the grid from that cell, adding rows as
 * needed.
 *
 * @example
 * <Input.Table value={rows} onChange={setRows}>
 *   <Input.TableColumn name="Pre-scaled" />
 *   <Input.TableColumn name="Scaled" />
 * </Input.Table>
 */
export declare const Table: ({ value, onChange, children, preview, rowLabel, createRow, className, ...rest }: TableProps) => ReactElement;
export {};
//# sourceMappingURL=Table.d.ts.map