// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { box } from "@synnaxlabs/x";
import {
  type ComponentPropsWithoutRef,
  forwardRef,
  type ReactElement,
  useEffect,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { table } from "@/table/aether";
import { Canvas } from "@/vis/canvas";

export interface TableProps
  extends ComponentPropsWithoutRef<"table">,
    Pick<z.infer<typeof table.Table.stateZ>, "visible"> {}

export const Table = Aether.wrap<TableProps>(
  "Table",
  ({ aetherKey, children, className, visible, ...props }): ReactElement => {
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: table.Table.TYPE,
      schema: table.Table.stateZ,
      initialState: {
        region: box.ZERO,
        visible,
      },
    });

    useEffect(() => {
      setState((s) => ({ ...s, visible }));
    }, [visible]);

    const ref = Canvas.useRegion((b) => setState((s) => ({ ...s, region: b })));

    return (
      <table ref={ref} className={CSS(CSS.B("table"), className)} {...props}>
        <tbody>
          <Aether.Composite path={path}>{children}</Aether.Composite>
        </tbody>
      </table>
    );
  },
);

export interface RowProps extends ComponentPropsWithoutRef<"tr"> {}

export const Row = ({ children, className, ...props }: RowProps): ReactElement => (
  <tr className={CSS(CSS.BE("table", "row"), className)} {...props}>
    {children}
  </tr>
);

export interface CellProps extends ComponentPropsWithoutRef<"td"> {
  selected?: boolean;
}

export const Cell = forwardRef<HTMLTableCellElement, CellProps>(
  (
    { children, className, selected = false, ...props }: CellProps,
    ref,
  ): ReactElement => (
    <td
      ref={ref}
      {...props}
      className={CSS(CSS.BE("table", "cell"), CSS.selected(selected), className)}
    >
      {children}
    </td>
  ),
);
Cell.displayName = "Cell";
