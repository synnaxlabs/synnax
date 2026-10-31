// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Cursor } from "@synnaxlabs/lyra/cursor";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { box, type direction, xy } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type PointerEvent as ReactPointerEvent,
  type ReactElement,
  type RefObject,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { INDICATOR_SIZE } from "@/table/Indicator";
import { transform } from "@/table/shift";

/** A move of rows or columns in progress. */
export interface Move {
  /** Indices of the rows or columns being moved, ascending. */
  indices: number[];
  /** Where the moved ones start once they land, counted among the rest. */
  to: number;
}

/** A row or column that a move can carry. */
export interface Item {
  key: string;
  size: number;
}

export interface UseMoveProps {
  /** x moves columns, y moves rows. */
  direction: direction.Direction;
  /** The rows or columns in display order. */
  items: Item[];
  /** Keys of the selected rows or columns. A press on one moves them all. */
  selected: string[];
  /** The table element. Rows and columns start one indicator in from its corner. */
  ref: RefObject<HTMLElement | null>;
  /**
   * Called on release with the finished move, resolved against the latest items. A
   * cancelled pointer drops the move.
   */
  onMove: (move: Move) => void;
}

export interface UseMoveReturn {
  /** The move in progress against the latest items, or null between moves. */
  move: Move | null;
  /** Handles a press on the indicator of the row or column with the given key. */
  press: (key: string, e: ReactPointerEvent) => void;
  /** Reports, once, whether the latest press became a move. */
  consumeMoved: () => boolean;
}

// Held is a move by the keys of what it carries, so it survives the items changing
// under it.
interface Held {
  keys: string[];
  to: number;
}

const offsetOf = (sizes: number[], index: number): number =>
  sizes.slice(0, index).reduce((a, s) => a + s, 0);

const indicesOf = (items: Item[], keys: string[]): number[] =>
  items.flatMap((item, i) => (keys.includes(item.key) ? [i] : []));

// resolve returns where the held items sit in items, or null when none are left.
const resolve = (items: Item[], { keys, to }: Held): Move | null => {
  const indices = indicesOf(items, keys);
  if (indices.length === 0) return null;
  return { indices, to: Math.min(to, items.length - indices.length) };
};

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
 * one. A press that never passes the drag threshold stays a click. The move follows its
 * rows or columns by key, so another session changing the table during the drag never
 * moves the wrong ones.
 */
export const useMove = ({
  direction: dir,
  items,
  selected,
  ref,
  onMove,
}: UseMoveProps): UseMoveReturn => {
  const [held, setHeld] = useState<Held | null>(null);
  const itemsRef = useSyncedRef(items);
  const selectedRef = useSyncedRef(selected);
  const onMoveRef = useSyncedRef(onMove);
  const heldRef = useRef<Held | null>(null);
  const pressedRef = useRef<{ key: string; keys: string[] }>({ key: "", keys: [] });
  // How far the pointer sits past the leading edge of the moved block.
  const grabRef = useRef(0);
  const movedRef = useRef(false);
  const startDrag = Cursor.useDrag({
    onStart: (start: xy.XY) => {
      const items = itemsRef.current;
      const { key, keys } = pressedRef.current;
      const pressed = items.findIndex((item) => item.key === key);
      if (ref.current == null || pressed === -1) return;
      movedRef.current = true;
      const sizes = items.map((item) => item.size);
      const indices = indicesOf(items, keys);
      const before = indices.filter((i) => i < pressed);
      const edge = offsetOf(sizes, pressed) - before.reduce((a, i) => a + sizes[i], 0);
      grabRef.current = start[dir] - originOf(ref.current, dir) - edge;
      heldRef.current = { keys, to: landing(sizes, indices, edge) };
      setHeld(heldRef.current);
      Cursor.setGlobalStyle("grabbing");
    },
    onMove: (_: box.Box, __: Triggers.Key, e: PointerEvent) => {
      const current = heldRef.current;
      if (ref.current == null || current == null) return;
      const items = itemsRef.current;
      const move = resolve(items, current);
      if (move == null) return;
      const pos = xy.construct(e)[dir] - originOf(ref.current, dir) - grabRef.current;
      const sizes = items.map((item) => item.size);
      const to = landing(sizes, move.indices, pos);
      if (to === current.to) return;
      heldRef.current = { keys: current.keys, to };
      setHeld(heldRef.current);
    },
    onEnd: (_: box.Box, __: Triggers.Key, e: PointerEvent) => {
      const current = heldRef.current;
      heldRef.current = null;
      Cursor.clearGlobalStyle();
      setHeld(null);
      if (current == null || e.type === "pointercancel") return;
      const move = resolve(itemsRef.current, current);
      if (move != null) onMoveRef.current(move);
    },
  });
  const press = (key: string, e: ReactPointerEvent) => {
    movedRef.current = false;
    const selected = selectedRef.current;
    pressedRef.current = { key, keys: selected.includes(key) ? selected : [key] };
    startDrag(e);
  };
  const consumeMoved = () => {
    const moved = movedRef.current;
    movedRef.current = false;
    return moved;
  };
  const move = held == null ? null : resolve(items, held);
  return { move, press, consumeMoved };
};

/**
 * Returns how far each row or column slides while a move previews where it lands: the
 * moved ones close up into a block at their landing, and the rest make room.
 */
export const shifts = (
  items: Array<{ size: number }>,
  { indices, to }: Move,
): number[] => {
  const sizes = items.map(({ size }) => size);
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
