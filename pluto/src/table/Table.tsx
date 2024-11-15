// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { ComponentPropsWithoutRef, ReactElement } from "react";

import { CSS } from "@/css";

export interface TableProps extends ComponentPropsWithoutRef<"table"> {}

export const Table = ({ children, className, ...props }: TableProps): ReactElement => {
  return (
    <table className={CSS(CSS.B("table"), className)} {...props}>
      <tbody>{children}</tbody>
    </table>
  );
};

export interface RowProps extends ComponentPropsWithoutRef<"tr"> {}

export const Row = ({ children, className, ...props }: RowProps): ReactElement => {
  return (
    <tr className={CSS(CSS.BE("table", "row"), className)} {...props}>
      {children}
    </tr>
  );
};

export interface CellProps extends ComponentPropsWithoutRef<"td"> {
  selected?: boolean;
}

export const Cell = ({
  children,
  className,
  selected = false,
  ...props
}: CellProps): ReactElement => {
  return (
    <td {...props} className={CSS(CSS.BE("table", "cell"), CSS.selected(selected))}>
      {children}
    </td>
  );
};
