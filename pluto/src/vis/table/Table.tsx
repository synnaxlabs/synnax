// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ComponentPropsWithoutRef, type ReactElement } from "react";

import { box, type direction } from "@synnaxlabs/x";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { type UseResizeHandler, useResize } from "@/hooks";
import { type telem } from "@/telem/aether";
import { table } from "@/vis/table/aether";

import "@/vis/table/Table.css";

export interface TableProps extends ComponentPropsWithoutRef<"table"> {
  numColumns: number;
}

export const Table = Aether.wrap<TableProps>(
  table.Table.TYPE,
  ({ aetherKey, numColumns, children, className, ...props }): ReactElement => {
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: table.Table.TYPE,
      schema: table.tableStateZ,
      initialState: {
        region: box.ZERO,
        widths: Array(numColumns).fill(0),
      },
    });

    const handleTableResize = useCallback<UseResizeHandler>(
      (b) => setState((p) => ({ ...p, region: b })),
      [setState],
    );

    const handleColumnResize = useCallback(
      (index: number, width: number) =>
        setState((p) => ({
          ...p,
          widths: p.widths.map((w, i) => (i === index ? width : w)),
        })),
      [setState],
    );

    const tableResizeRef = useResize(handleTableResize);
    return (
      <table ref={tableResizeRef} className={CSS(CSS.B("table"), className)} {...props}>
        <tbody>
          <Aether.Composite path={path}>{children}</Aether.Composite>
          <tr className={CSS.BE("table", "virtual-row")}>
            {Array(numColumns)
              .fill(0)
              .map((_, i) => (
                <VirtualTD key={i} index={i} onResize={handleColumnResize} />
              ))}
          </tr>
        </tbody>
      </table>
    );
  },
);

export interface VirtualTDProps {
  index: number;
  onResize: (index: number, width: number) => void;
}

const VirtualTD = ({ onResize, index }: VirtualTDProps): ReactElement => {
  const handleResize = useCallback(
    (b: box.Box) => onResize(index, box.width(b)),
    [onResize, index],
  );
  const ref = useResize(handleResize);
  return <td ref={ref} className={CSS.BE("table", "virtual-cell")} />;
};

export interface TRProps extends ComponentPropsWithoutRef<"tr"> {}

const TR_RESIZE_TRIGGERS: direction.Direction[] = ["y"];

export const TR = Aether.wrap<TRProps>(
  table.TR.TYPE,
  ({ aetherKey, children }): ReactElement => {
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: table.TR.TYPE,
      schema: table.trStateZ,
      initialState: {
        height: 0,
      },
    });

    const handleResize: UseResizeHandler = useCallback(
      (b) => setState((p) => ({ ...p, height: box.height(b) })),
      [setState],
    );

    const ref = useResize(handleResize, { triggers: TR_RESIZE_TRIGGERS });

    return (
      <tr ref={ref}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </tr>
    );
  },
);

export interface StringTDProps
  extends z.input<typeof table.stringTDStateZ>,
    Omit<ComponentPropsWithoutRef<"td">, "children" | "color"> {
  telem: telem.StringSourceSpec;
}

export const StringTD = Aether.wrap<StringTDProps>(
  table.StringTD.TYPE,
  ({ aetherKey, telem }): ReactElement => {
    Aether.use({
      aetherKey,
      type: table.StringTD.TYPE,
      schema: table.stringTDStateZ,
      initialState: {
        stringSource: telem,
      },
    });
    return <td style={{ height: 20 }}></td>;
  },
);
