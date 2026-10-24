// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/lyra/css";
import { Cursor } from "@synnaxlabs/lyra/cursor";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { box, type direction, xy } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type PointerEvent as ReactPointerEvent,
  type ReactElement,
  type RefObject,
  useCallback,
  useRef,
  useState,
} from "react";

import { INDICATOR_SIZE } from "@/table/Indicator";
import { transform } from "@/table/shift";

/** A move of rows or columns in progress. */
export interface Move {
  /** Indices of the rows or columns being moved, ascending. */
  indices: number[];
  /** Where the moved ones start once they land, counted among the rest. */
  to: number;
}

export interface UseMoveProps {
  /** x moves columns, y moves rows. */
  direction: direction.Direction;
  /** Sizes of the rows or columns in display order. */
  sizes: number[];
  /** Indices of the selected rows or columns. A press on one moves them all. */
  selected: number[];
  /** The table element. Rows and columns start one indicator in from its corner. */
  ref: RefObject<HTMLElement | null>;
  /** Called on release with the finished move. A cancelled pointer drops the move. */
  onMove: (move: Move) => void;
}

export interface UseMoveReturn {
  /** The move in progress, or null between moves. */
  move: Move | null;
  /** Handles a press on the indicator of the row or column at index. */
  press: (index: number, e: ReactPointerEvent) => void;
  /** Reports, once, whether the latest press became a move. */
  consumeMoved: () => boolean;
}

const offsetOf = (sizes: number[], index: number): number =>
  sizes.slice(0, index).reduce((a, s) => a + s, 0);

// landing returns where the moved items land when their leading edge sits at pos: the
// number of the rest whose midpoint, with the moved ones lifted out, lies before pos.
const landing = (sizes: number[], indices: number[], pos: number): number => {
  let to = 0;
  let edge = 0;
  for (const [i, size] of sizes.entries()) {
    if (indices.includes(i)) continue;
    if (pos <= edge + size / 2) break;
    edge += size;
    to++;
  }
  return to;
};

// originOf returns where the cells of the table el start along dir. Callers measure on
// every event, so a scroll during the drag keeps the edges in place. The pressed
// indicator slides with the preview, so it is no fixed reference.
const originOf = (el: HTMLElement, dir: direction.Direction): number =>
  box.topLeft(box.construct(el))[dir] + INDICATOR_SIZE;

/**
 * Drives moving rows or columns by dragging an indicator. A press on a selected row or
 * column moves every selected one as a block, and a press elsewhere moves just that
 * one. A press that never passes the drag threshold stays a click.
 */
export const useMove = ({
  direction: dir,
  sizes,
  selected,
  ref,
  onMove,
}: UseMoveProps): UseMoveReturn => {
  const [move, setMove] = useState<Move | null>(null);
  const sizesRef = useSyncedRef(sizes);
  const selectedRef = useSyncedRef(selected);
  const onMoveRef = useSyncedRef(onMove);
  const moveRef = useRef<Move>({ indices: [], to: 0 });
  const pressedRef = useRef(0);
  // How far the pointer sits past the leading edge of the moved block.
  const grabRef = useRef(0);
  const movedRef = useRef(false);
  const startDrag = Cursor.useDrag({
    onStart: useCallback(
      (start: xy.XY) => {
        if (ref.current == null) return;
        movedRef.current = true;
        const sizes = sizesRef.current;
        const { indices } = moveRef.current;
        const pressed = pressedRef.current;
        const before = indices.filter((i) => i < pressed);
        const edge =
          offsetOf(sizes, pressed) - before.reduce((a, i) => a + sizes[i], 0);
        grabRef.current = start[dir] - originOf(ref.current, dir) - edge;
        moveRef.current = { indices, to: landing(sizes, indices, edge) };
        setMove(moveRef.current);
        Cursor.setGlobalStyle("grabbing");
      },
      [dir, ref],
    ),
    onMove: useCallback(
      (_: box.Box, __: Triggers.Key, e: PointerEvent) => {
        if (ref.current == null) return;
        const { indices, to: prev } = moveRef.current;
        const pos = xy.construct(e)[dir] - originOf(ref.current, dir) - grabRef.current;
        const to = landing(sizesRef.current, indices, pos);
        if (to === prev) return;
        moveRef.current = { indices, to };
        setMove(moveRef.current);
      },
      [dir, ref],
    ),
    onEnd: useCallback((_: box.Box, __: Triggers.Key, e: PointerEvent) => {
      Cursor.clearGlobalStyle();
      setMove(null);
      if (e.type !== "pointercancel") onMoveRef.current(moveRef.current);
    }, []),
  });
  const press = useCallback(
    (index: number, e: ReactPointerEvent) => {
      movedRef.current = false;
      pressedRef.current = index;
      const selected = selectedRef.current;
      moveRef.current = {
        indices: selected.includes(index) ? selected : [index],
        to: 0,
      };
      startDrag(e);
    },
    [startDrag],
  );
  const consumeMoved = useCallback(() => {
    const moved = movedRef.current;
    movedRef.current = false;
    return moved;
  }, []);
  return { move, press, consumeMoved };
};

/**
 * Returns how far each row or column slides while a move previews where it lands: the
 * moved ones close up into a block at their landing, and the rest make room.
 */
export const shifts = (sizes: number[], { indices, to }: Move): number[] => {
  const rest = [...sizes.keys()].filter((i) => !indices.includes(i));
  const order = [...rest.slice(0, to), ...indices, ...rest.slice(to)];
  const result = new Array<number>(sizes.length);
  let offset = 0;
  for (const i of order) {
    result[i] = offset - offsetOf(sizes, i);
    offset += sizes[i];
  }
  return result;
};

/**
 * Returns the single moves that carry the moved rows or columns, kept in order, into
 * one block at their landing. Forward moves go last to first and backward moves first
 * to last, so no step shifts an index that a later step names.
 */
export const steps = ({ indices, to }: Move): Array<{ from: number; to: number }> => {
  const all = indices.map((from, k) => ({ from, to: to + k }));
  return [
    ...all.filter((s) => s.from < s.to).reverse(),
    ...all.filter((s) => s.from > s.to),
  ];
};

export interface OverlayProps {
  direction: direction.Direction;
  move: Move;
  sizes: number[];
  /** The shifts of the rows or columns, from {@link shifts}. */
  shifts: number[];
  /** Size of the table along the other axis, without the indicator strip. */
  span: number;
}

/**
 * Shades the moving rows or columns, sliding each into its landing slot. Positions are
 * relative to the table's top left corner.
 */
export const Overlay = ({
  direction: dir,
  move: { indices },
  sizes,
  shifts,
  span,
}: OverlayProps): ReactElement => {
  const length = INDICATOR_SIZE + span;
  return (
    <>
      {indices.map((i) => {
        const pos = INDICATOR_SIZE + offsetOf(sizes, i);
        const style: CSSProperties = {
          ...(dir === "y"
            ? { top: pos, left: 0, height: sizes[i], width: length }
            : { left: pos, top: 0, width: sizes[i], height: length }),
          transform: transform(xy.construct(dir, shifts[i])),
        };
        return (
          <div key={i} className={CSS.BE("table-frame", "move-source")} style={style} />
        );
      })}
    </>
  );
};
