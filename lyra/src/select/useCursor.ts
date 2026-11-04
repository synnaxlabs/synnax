// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type destructor,
  type location,
  type record,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import { type KeyboardEvent, useEffect, useRef, useState } from "react";

import { Dialog } from "@/dialog";
import { useSyncedRef } from "@/hooks";
import { List } from "@/list";
import { Triggers } from "@/triggers";
import { isInputOrContentEditable } from "@/util/event";

/**
 * What the arrow keys of a selection move. A hover is a highlight that the pointer
 * shares and Enter clicks, and it answers keys anywhere in the window. The selection
 * moves itself, Enter double-clicks it, and it answers keys only on the focused element
 * that spreads the handlers of `Select.useKeyboard`.
 */
export type Cursor = "hover" | "selection";

/** Reads and acts on the options of the enclosing frame by key. */
export interface Order<K extends record.Key> {
  /** Changes identity whenever the data changes. */
  getOrder: () => K[];
  scrollTo: (key: K, direction: location.Y) => void;
  /** Clicks the option with the given key, as a mouse click would. */
  click: (key: K) => void;
  /** Double-clicks the option with the given key, as a mouse would. */
  doubleClick: (key: K) => void;
  /** Calls the listener whenever the fixed items change. */
  subscribe: (listener: () => void) => destructor.Destructor;
}

/** Props for {@link useCursor}. */
export interface UseCursorProps<K extends record.Key> {
  /** Position in the order a hover starts on. Defaults to none. */
  initialHover?: number;
  /** What the arrow keys move. Defaults to a hover. */
  cursor?: Cursor;
  /**
   * When to answer the keys. Defaults to the enclosing dialog's visibility for a hover,
   * and to always for the selection.
   */
  enableTriggers?: Triggers.Condition;
  /**
   * Returns the key that Left or Right moves the cursor to from the given key, or
   * undefined to stay. It may change the options instead, as a tree expands a node.
   * Left and Right do nothing without it.
   */
  moveX?: (key: K, direction: location.X) => K | undefined;
  order: Order<K>;
  /** The selection cursor starts on the first of these keys in the order. */
  selected: K[];
  /** Replaces the selection with the given key. */
  select: (key: K) => void;
}

/** Handlers for the focused element whose keys move a selection cursor. */
export interface KeyboardHandlers {
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
  onKeyUp: (e: KeyboardEvent<HTMLElement>) => void;
  onBlur: () => void;
}

/** Return value for {@link useCursor}. */
export interface UseCursorReturn<K extends record.Key> {
  /** The key a hover rests on. Always undefined when the keys move the selection. */
  hover: K | undefined;
  /**
   * Moves a hover onto the option under the pointer, so the pointer and the arrow keys
   * share one hover. Does nothing while the keys are off, or when the pointer has not
   * moved since its last move.
   */
  setHover: (key: K) => void;
  /** Moves a selection cursor onto a clicked key. A hover follows the pointer. */
  setCursor: (key: K) => void;
  keyboard: KeyboardHandlers;
}

type CursorKey =
  "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight" | "Home" | "End" | "Enter";

const Y_KEYS: CursorKey[] = ["ArrowUp", "ArrowDown", "Home", "End", "Enter"];
const KEYS: CursorKey[] = [...Y_KEYS, "ArrowLeft", "ArrowRight"];
const KEY_SET: ReadonlySet<string> = new Set(KEYS);
// A text field moves its caret with these, so they stay with the field.
const CARET_KEYS: ReadonlySet<CursorKey> = new Set([
  "ArrowLeft",
  "ArrowRight",
  "Home",
  "End",
]);
const Y_TRIGGERS: Triggers.Trigger[] = Y_KEYS.map((k) => [k]);
const TRIGGERS: Triggers.Trigger[] = KEYS.map((k) => [k]);

const isCursorKey = (key: string | null): key is CursorKey =>
  key != null && KEY_SET.has(key);

const isX = (key: CursorKey): key is "ArrowLeft" | "ArrowRight" =>
  key === "ArrowLeft" || key === "ArrowRight";

const REPEAT_DELAY = TimeSpan.milliseconds(200).milliseconds;
const REPEAT_INTERVAL = TimeSpan.milliseconds(100).milliseconds;

// The index a vertical key moves the cursor to from index at, where -1 is no cursor. A
// held key stops at the ends, and a new press wraps around.
const targetIndex = (
  key: CursorKey,
  at: number,
  count: number,
  held: boolean,
): number | null => {
  const last = count - 1;
  if (key === "Home") return 0;
  if (key === "End") return last;
  if (key === "ArrowDown") {
    if (at === -1) return 0;
    if (at < last) return at + 1;
    return held ? null : 0;
  }
  if (at === -1) return last;
  if (at > 0) return at - 1;
  return held ? null : last;
};

// Where the cursor rests when the key it last moved to is not among the options.
const startOf = <K extends record.Key>(
  order: K[],
  cursor: Cursor,
  initialHover: number,
  selected: K[],
): K | undefined => {
  if (cursor === "selection") {
    if (selected.length === 0) return undefined;
    const keys = new Set(selected);
    return order.find((k) => keys.has(k));
  }
  if (initialHover < 0 || order.length === 0) return undefined;
  return order[initialHover >= order.length ? 0 : initialHover];
};

/**
 * Moves a cursor through the options of the enclosing frame with the arrow keys, Home,
 * and End, and acts on it with Enter. The cursor is a hover or the selection itself;
 * see {@link Cursor}. Holding a key repeats it, and the frame scrolls the cursor into
 * view and keeps it mounted. The cursor is a key, so it survives options appearing or
 * disappearing around it.
 */
export const useCursor = <K extends record.Key>({
  initialHover = -1,
  cursor = "hover",
  enableTriggers,
  moveX,
  order: { getOrder, subscribe, scrollTo, click, doubleClick },
  selected,
  select,
}: UseCursorProps<K>): UseCursorReturn<K> => {
  const hovering = cursor === "hover";
  // The key the cursor last moved to, which may since have left the options.
  const movedRef = useRef<K | undefined>(undefined);
  const [current, setCurrent] = useState<K | undefined>(undefined);
  const resolve = (order: K[]): K | undefined => {
    const moved = movedRef.current;
    if (moved != null && order.includes(moved)) return moved;
    return startOf(order, cursor, initialHover, selected);
  };
  useEffect(() => {
    const update = (): void => setCurrent(resolve(getOrder()));
    update();
    return subscribe(update);
  }, [subscribe, resolve, getOrder]);
  List.usePin(current);
  const { visible } = Dialog.useContext();
  const enabledRef = useSyncedRef<Triggers.Condition>(
    enableTriggers ?? (hovering ? visible : true),
  );

  const moveTo = (key: K, direction: location.Y): void => {
    movedRef.current = key;
    setCurrent(key);
    if (!hovering) select(key);
    scrollTo(key, direction);
  };
  const step = (key: CursorKey, held: boolean): void => {
    const order = getOrder();
    if (order.length === 0) return;
    const from = resolve(order);
    const at = from == null ? -1 : order.indexOf(from);
    let to: K | undefined;
    if (isX(key)) {
      if (from != null) to = moveX?.(from, key === "ArrowLeft" ? "left" : "right");
    } else {
      const index = targetIndex(key, at, order.length, held);
      if (index != null) to = order[index];
    }
    if (to == null || to === from) return;
    moveTo(to, order.indexOf(to) < at ? "bottom" : "top");
  };
  const activate = (): void => {
    const key = resolve(getOrder());
    if (key == null) return;
    if (hovering) click(key);
    else doubleClick(key);
  };
  // The repeat timer outlives the render that started it, so it reads the latest step.
  const stepRef = useSyncedRef(step);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const release = (): void => {
    if (timerRef.current != null) clearInterval(timerRef.current);
    timerRef.current = null;
  };
  useEffect(
    () => () => {
      if (timerRef.current != null) clearInterval(timerRef.current);
    },
    [],
  );
  const press = (key: CursorKey): void => {
    release();
    if (key === "Enter") return activate();
    stepRef.current(key, false);
    timerRef.current = setTimeout(() => {
      timerRef.current = setInterval(() => stepRef.current(key, true), REPEAT_INTERVAL);
    }, REPEAT_DELAY);
  };

  const handleTrigger = ({ triggers, stage, target }: Triggers.UseEvent): void => {
    if (stage === "end") return release();
    const key = KEYS.find((k) => triggers.some((t) => t.includes(k)));
    if (key == null || (CARET_KEYS.has(key) && isInputOrContentEditable({ target })))
      return;
    press(key);
  };
  Triggers.use({
    triggers: moveX == null ? Y_TRIGGERS : TRIGGERS,
    callback: handleTrigger,
    loose: true,
    enabled: () => hovering && Triggers.resolveCondition(enabledRef.current),
  });
  const keyboard: KeyboardHandlers = {
    onKeyDown: (e) => {
      // Keys typed into an element inside the list, such as a name being edited, keep
      // their own meaning.
      if (hovering || e.defaultPrevented || e.target !== e.currentTarget) return;
      if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
      const key = Triggers.eventKey(e);
      if (!isCursorKey(key) || (isX(key) && moveX == null)) return;
      if (!Triggers.resolveCondition(enabledRef.current)) return;
      e.preventDefault();
      // The cursor repeats a held key at its own pace.
      if (!e.repeat) press(key);
    },
    onKeyUp: (e) => {
      if (isCursorKey(Triggers.eventKey(e))) release();
    },
    onBlur: release,
  };

  // WebKit sends a pointer move when the list scrolls or appears under a still pointer.
  // Only a move to a new position moves the hover, or the arrow keys would scroll the
  // hover back under the pointer.
  const pointerMovedRef = useRef(false);
  const tracking =
    hovering && (typeof enableTriggers === "function" || (enableTriggers ?? visible));
  useEffect(() => {
    if (!tracking) return;
    let last: xy.XY | undefined;
    const handleMove = (e: PointerEvent): void => {
      const position = xy.construct(e);
      pointerMovedRef.current = last != null && !xy.equals(last, position);
      last = position;
    };
    window.addEventListener("pointermove", handleMove, {
      capture: true,
      passive: true,
    });
    return () =>
      window.removeEventListener("pointermove", handleMove, { capture: true });
  }, [tracking]);
  const place = (key: K): void => {
    movedRef.current = key;
    setCurrent(key);
  };
  const setHover = (key: K): void => {
    if (
      movedRef.current === key ||
      !pointerMovedRef.current ||
      !Triggers.resolveCondition(enabledRef.current)
    )
      return;
    place(key);
  };
  const setCursor = (key: K): void => {
    if (!hovering) place(key);
  };
  return { hover: hovering ? current : undefined, setHover, setCursor, keyboard };
};
