// Copyright 2025 Synnax Labs, Inc.
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
  type ComponentPropsWithRef,
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
import { Menu } from "@/menu";
import { table } from "@/table/aether";
import { Text } from "@/text";
import { stopPropagation } from "@/util/event";
import { Canvas } from "@/vis/canvas";

export interface TableProps
  extends ComponentPropsWithoutRef<"table">,
    Pick<z.infer<typeof table.Table.stateZ>, "visible"> {}

export const Table = ({
  children,
  className,
  visible,
  ...rest
}: TableProps): ReactElement => {
  const [{ path }, , setState] = Aether.use({
    type: table.Table.TYPE,
    schema: table.Table.stateZ,
    initialState: { region: box.ZERO, visible },
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
      />
      <table className={CSS(CSS.B("table"), className)} {...rest}>
        <tbody>
          <Aether.Composite path={path}>{children}</Aether.Composite>
        </tbody>
      </table>
    </>
  );
};

export interface RowProps
  extends Omit<ComponentPropsWithoutRef<"tr">, "size" | "onResize" | "onSelect"> {
  index: number;
  size: number;
  position: number;
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
  position,
  ...rest
}: RowProps): ReactElement => (
  <tr className={CSS(CSS.BE("table", "row"), className)} {...rest}>
    {onResize != null && (
      <Indicator
        onSelect={onSelect}
        index={index}
        value={size}
        onChange={onResize}
        position={position}
        direction="y"
      />
    )}
    {children}
  </tr>
);

export interface CellProps extends ComponentPropsWithRef<"td"> {
  selected?: boolean;
}

export const Cell = ({
  ref,
  children,
  className,
  selected = false,
  ...rest
}: CellProps): ReactElement => (
  <td
    ref={ref}
    {...rest}
    className={CSS(CSS.BE("table", "cell"), CSS.selected(selected), className)}
  >
    {children}
  </td>
);

interface ResizerCellProps {
  direction: direction.Direction;
  value: number;
  index: number;
  selected?: boolean;
  position: number;
  onChange: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

export const getCellColumn = (index: number): string => ALPHABET[index];

const Indicator = ({
  value,
  selected = false,
  index,
  onChange,
  position,
  direction: dir = "x",
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
      id={`resizer-${dir}-${index}`}
      className={CSS(
        CSS.BE("table", "resizer"),
        CSS.dir(dir),
        CSS.selected(selected),
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
      )}
      style={{ [direction.dimension(dir)]: value }}
      onClick={() => onSelect(index)}
      onContextMenu={() => onSelect(index)}
    >
      <Text.Text full="x" justify="center" align="center" square={false}>
        {dir === "x" ? ALPHABET[index] : index + 1}
      </Text.Text>
      <button
        onClick={stopPropagation}
        style={{ [direction.location(dir)]: position + value }}
        onDragStart={onDragStart}
        draggable
      />
    </td>
  );
};

interface ColumnIndicators {
  selected: number[];
  columns: number[];
  onResize: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

export const ColumnIndicators = ({
  selected,
  onSelect,
  columns,
  onResize,
}: ColumnIndicators) => {
  let currPos = 2.5 * 6;
  return (
    <tr className={CSS(CSS.BE("table", "row"), CSS.BE("table", "col-resizer"))}>
      <td />
      {columns.map((size, i) => {
        const pos = currPos;
        currPos += size;
        return (
          <Indicator
            onSelect={onSelect}
            key={i}
            position={pos}
            selected={selected.includes(i)}
            index={i}
            value={size}
            onChange={(size) => onResize(size, i)}
            direction="x"
          />
        );
      })}
    </tr>
  );
};
