// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { box, direction } from "@synnaxlabs/x";
import {
  type ComponentPropsWithoutRef,
  forwardRef,
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { table } from "@/table/aether";
import { Text } from "@/text";
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
      <>
        <div
          ref={ref}
          style={{
            right: 0,
            bottom: 0,
            position: "absolute",
            top: 6,
            left: 6,
          }}
        ></div>
        <table className={CSS(CSS.B("table"), className)} {...props}>
          <tbody>
            <Aether.Composite path={path}>{children}</Aether.Composite>
          </tbody>
        </table>
      </>
    );
  },
);

export interface RowProps
  extends Omit<ComponentPropsWithoutRef<"tr">, "size" | "onResize" | "onSelect"> {
  index: number;
  size: number;
  onResize?: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

export const Row = ({
  children,
  className,
  size,
  index,
  onResize,
  onSelect,
  ...props
}: RowProps): ReactElement => (
  <tr className={CSS(CSS.BE("table", "row"), className)} {...props}>
    {onResize != null && (
      <ResizerCell
        onSelect={onSelect}
        index={index}
        value={size}
        onChange={onResize}
        direction="y"
      />
    )}
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

interface ResizerCellProps {
  direction: direction.Direction;
  value: number;
  index: number;
  selected?: boolean;
  onChange: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

const ResizerCell = ({
  value,
  selected = false,
  index,
  onChange,
  direction: dir,
  onSelect,
}: ResizerCellProps) => {
  const valueRef = useSyncedRef(value);
  const sizeRef = useRef(value);
  const onDragStart = useCursorDrag({
    onStart: useCallback(() => {
      sizeRef.current = valueRef.current;
    }, []),
    onMove: useCallback(
      (b: box.Box) => onChange(sizeRef.current + box.dim(b, dir, true), index),
      [onChange, index],
    ),
  });

  return (
    <td
      onDragStart={onDragStart}
      className={CSS(CSS.BE("table", "resizer"), CSS.dir(dir), CSS.selected(selected))}
      style={{ [direction.dimension(dir)]: value }}
      draggable
      onClick={() => onSelect(index)}
      onContextMenu={() => onSelect(index)}
    >
      <Text.Text level="p" shade={7} style={{ width: "100%", textAlign: "center" }}>
        {dir === "x" ? ALPHABET[index] : index + 1}
      </Text.Text>
      <button />
    </td>
  );
};

interface ColResizerProps {
  selected: number[];
  columns: number[];
  onResize: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

export const ColResizer = ({
  selected,
  onSelect,
  columns,
  onResize,
}: ColResizerProps) => (
  <tr className={CSS(CSS.BE("table", "row"), CSS.BE("table", "col-resizer"))}>
    <td></td>
    {columns.map((size, i) => (
      <ResizerCell
        onSelect={onSelect}
        key={i}
        selected={selected.includes(i)}
        index={i}
        value={size}
        onChange={(size) => onResize(size, i)}
        direction="x"
      />
    ))}
  </tr>
);
