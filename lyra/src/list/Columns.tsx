// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Columns.css";

import {
  type ComponentPropsWithRef,
  type PropsWithChildren,
  type ReactElement,
} from "react";

import { context } from "@/context";
import { CSS } from "@/css";

/** A column that the items of a list line up on. */
export interface Column {
  /** The heading. A column with no name, such as one for an icon, has none. */
  name?: string;
  /**
   * The width as a grid track size, such as "30rem" or "1fr". Each item is its own
   * grid, so a width that follows the content, such as "auto", does not line up.
   */
  width: string;
  /** Whether the column's cells hold fields, so its heading lines up with their text. */
  field?: boolean;
}

const [Context, useContext] = context.create<Column[]>({
  displayName: "List.Columns",
  providerName: "List.Columns",
});

/** Props for {@link Columns}. */
export interface ColumnsProps extends PropsWithChildren {
  columns: Column[];
}

/** Declares the columns that {@link Headings} and every {@link Cells} share. */
export const Columns = ({ columns, children }: ColumnsProps): ReactElement => (
  <Context value={columns}>{children}</Context>
);

/** Props for {@link Cells}. */
export interface CellsProps extends ComponentPropsWithRef<"div"> {}

/** Lays out its children on the declared columns, one child per column. */
export const Cells = ({ className, style, ...rest }: CellsProps): ReactElement => {
  const columns = useContext("List.Cells");
  return (
    <div
      className={CSS.cls(CSS.BE("list", "cells"), className)}
      style={{ ...style, gridTemplateColumns: columns.map((c) => c.width).join(" ") }}
      {...rest}
    />
  );
};

/** Props for {@link Cell}. */
export interface CellProps extends ComponentPropsWithRef<"div"> {}

/**
 * One column of an item. Its child fills the column. A cell with no child is the empty
 * cell of a column that does not apply to the item.
 */
export const Cell = ({ className, ...rest }: CellProps): ReactElement => (
  <div className={CSS.cls(CSS.BE("list", "cell"), className)} {...rest} />
);

/** Props for {@link Headings}. */
export interface HeadingsProps {
  /** Whether the items lead with a checkbox, which the headings leave room for. */
  checkable?: boolean;
}

/** The names of the declared columns, on a band above the items. */
export const Headings = ({ checkable = false }: HeadingsProps): ReactElement => {
  const columns = useContext("List.Headings");
  return (
    <div className={CSS.BE("list", "headings")}>
      {checkable && <span className={CSS.BE("list", "headings-check")} />}
      <Cells>
        {columns.map(({ name, field = false }, i) => (
          <span
            key={i}
            className={CSS.cls(field && CSS.BEM("list", "heading", "field"))}
          >
            {name}
          </span>
        ))}
      </Cells>
    </div>
  );
};
