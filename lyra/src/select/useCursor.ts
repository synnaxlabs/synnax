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
import { useDestructors, useSyncedRef } from "@/hooks";
import { Triggers } from "@/triggers";
import { isInputOrContentEditable, moveIndex } from "@/util";

/**
 * What the arrow keys of a selection move. A hover is a highlight that the pointer
 * shares and Enter clicks, and it answers keys anywhere in the window. The selection
 * moves itself, Enter double-clicks it, Shift+F10 and the Menu key open its context
 * menu, and it answers keys only on the focused element that spreads the handlers of
 * `Select.useKeyboard`.
 */
export type Cursor = "hover" | "selection";

/** Reads and acts on the options by key. */
export interface Order<K extends record.Key> {
  /**
   * Reads the options in order. When they change, it changes identity or the listeners
   * of subscribe run.
   */
  getOrder: () => K[];
  scrollTo: (key: K, direction: location.Y) => void;
  /** Clicks the option with the given key, as a mouse click would. */
  click: (key: K) => void;
  /** Calls the listener whenever the options change but getOrder keeps its identity. */
  subscribe: (listener: () => void) => destructor.Destructor;
}

/** An order whose options a selection cursor also opens and shows menus for. */
export interface SelectionOrder<K extends record.Key> extends Order<K> {
  /** Double-clicks the option with the given key, as a mouse would. */
  doubleClick: (key: K) => void;
  /** Opens the context menu of the option with the given key, under the option. */
  contextMenu: (key: K) => void;
}

interface BaseProps<K extends record.Key> {
  /** Position in the order a hover starts on. Defaults to none. */
  initialHover?: number;
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
}

interface HoverProps<K extends record.Key> extends BaseProps<K> {
  cursor?: "hover";
  order: Order<K>;
}

interface SelectionProps<K extends record.Key> extends BaseProps<K> {
  cursor: "selection";
  order: SelectionOrder<K>;
  /** The cursor starts on the first of these keys in the order. */
  selected: K[];
  /** Replaces the selection with the given key. */
  select: (key: K) => void;
}

/** Props for {@link useCursor}. The cursor defaults to a hover. */
export type UseCursorProps<K extends record.Key> = HoverProps<K> | SelectionProps<K>;

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
  /**
   * The key Enter and the menu keys act on. A list keeps it mounted with `List.usePin`,
   * so the keys reach it out of view.
   */
  target: K | undefined;
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

// Shift+F10 opens the context menu on its press, and the Menu key on its release, as
// on Windows. Both keep the webview from opening its own.
const menuKey = (e: KeyboardEvent<HTMLElement>): "F10" | "ContextMenu" | null => {
  if (e.altKey || e.ctrlKey || e.metaKey) return null;
  const key = Triggers.eventKey(e);
  if (key === "ContextMenu" || (key === "F10" && e.shiftKey)) return key;
  return null;
};

const REPEAT_DELAY = TimeSpan.milliseconds(200).milliseconds;
const REPEAT_INTERVAL = TimeSpan.milliseconds(100).milliseconds;

// Where the cursor rests when the key it last moved to is not among the options. A
// hover has no selection.
const startOf = <K extends record.Key>(
  order: K[],
  initialHover: number,
  selected: K[] | undefined,
): K | undefined => {
  if (selected != null) {
    if (selected.length === 0) return undefined;
    const keys = new Set(selected);
    return order.find((k) => keys.has(k));
  }
  if (initialHover < 0 || order.length === 0) return undefined;
  return order[initialHover >= order.length ? 0 : initialHover];
};

/**
 * Moves a cursor through the options with the arrow keys, Home, and End, and acts on it
 * with Enter and the menu keys. The cursor is a hover or the selection itself; see
 * {@link Cursor}. Holding a key repeats it, and the order scrolls the cursor into view.
 * The cursor is a key, so it survives options appearing or disappearing around it.
 */
export const useCursor = <K extends record.Key>(
  props: UseCursorProps<K>,
): UseCursorReturn<K> => {
  const {
    initialHover = -1,
    enableTriggers,
    moveX,
    order: { getOrder, subscribe, scrollTo, click },
  } = props;
  const selection = props.cursor === "selection" ? props : null;
  const hovering = selection == null;
  // The key the cursor last moved to, which may since have left the options.
  const movedRef = useRef<K | undefined>(undefined);
  // The selection when the cursor last moved. A selection that has since gained a key
  // changed without the cursor, as when a duplicate selects its copy, so the cursor
  // follows it. One that only lost keys keeps a key a modifier click deselected.
  const movedFromRef = useRef<K[]>([]);
  const follows = (moved: K): boolean =>
    selection == null ||
    selection.selected.includes(moved) ||
    selection.selected.every((k) => movedFromRef.current.includes(k));
  const resolve = (order: K[]): K | undefined => {
    const moved = movedRef.current;
    if (moved != null && order.includes(moved) && follows(moved)) return moved;
    return startOf(order, initialHover, selection?.selected);
  };
  // A modifier click can deselect the key the cursor rests on, and Enter and the menu
  // keys then act on the selection.
  const targetOf = (order: K[]): K | undefined => {
    const key = resolve(order);
    if (key == null || selection == null || selection.selected.includes(key))
      return key;
    return startOf(order, initialHover, selection.selected);
  };
  const [target, setTarget] = useState<K | undefined>(undefined);
  useEffect(() => {
    const update = (): void => setTarget(targetOf(getOrder()));
    update();
    return subscribe(update);
  }, [subscribe, targetOf, getOrder]);
  const { visible } = Dialog.useContext();
  const enabledRef = useSyncedRef<Triggers.Condition>(
    enableTriggers ?? (hovering ? visible : true),
  );

  const moveTo = (key: K, direction: location.Y): void => {
    movedRef.current = key;
    setTarget(key);
    if (selection != null) {
      movedFromRef.current = selection.selected;
      selection.select(key);
    }
    scrollTo(key, direction);
  };
  const step = (key: Exclude<CursorKey, "Enter">, held: boolean): void => {
    const order = getOrder();
    if (order.length === 0) return;
    const from = resolve(order);
    const at = from == null ? -1 : order.indexOf(from);
    let to: K | undefined;
    if (isX(key)) {
      if (from != null) to = moveX?.(from, key === "ArrowLeft" ? "left" : "right");
    } else {
      const index = moveIndex(key, at, order.length, held);
      if (index != null) to = order[index];
    }
    if (to == null || to === from) return;
    moveTo(to, order.indexOf(to) < at ? "bottom" : "top");
  };
  const activate = (): void => {
    const key = targetOf(getOrder());
    if (key == null) return;
    if (selection == null) click(key);
    else selection.order.doubleClick(key);
  };
  const openMenu = (): void => {
    const key = targetOf(getOrder());
    if (key != null) selection?.order.contextMenu(key);
  };
  // The repeat outlives the render that started it, so it reads the latest step.
  const stepRef = useSyncedRef(step);
  const repeat = useDestructors();
  const release = repeat.cleanup;
  const heldRef = useRef(false);
  const press = (key: CursorKey): void => {
    release();
    if (key === "Enter") return activate();
    stepRef.current(key, false);
    // A late release is no hold, so a selection repeats only once the OS repeats the
    // key. A hover reads the keys through Triggers, which drops the repeats.
    heldRef.current = hovering;
    let timer = setTimeout(() => {
      timer = setInterval(() => {
        if (heldRef.current) stepRef.current(key, true);
      }, REPEAT_INTERVAL);
    }, REPEAT_DELAY);
    // A window that loses focus while a key is held never sees the release.
    const stop = (): void => {
      clearInterval(timer);
      window.removeEventListener("blur", stop);
    };
    window.addEventListener("blur", stop);
    repeat.set(stop);
  };

  const handleTrigger = (e: Triggers.UseEvent): void => {
    if (e.stage === "end") return release();
    const key = KEYS.find((k) => e.triggers.some((t) => t.includes(k)));
    if (key == null || (CARET_KEYS.has(key) && isInputOrContentEditable(e))) return;
    press(key);
  };
  Triggers.use({
    triggers: moveX == null ? Y_TRIGGERS : TRIGGERS,
    callback: handleTrigger,
    loose: true,
    enabled: () => hovering && Triggers.resolveCondition(enabledRef.current),
  });
  // Keys typed into an element inside the list, such as a name being edited, keep their
  // own meaning.
  const answers = (e: KeyboardEvent<HTMLElement>): boolean =>
    !hovering &&
    !e.defaultPrevented &&
    e.target === e.currentTarget &&
    Triggers.resolveCondition(enabledRef.current);
  const keyboard: KeyboardHandlers = {
    onKeyDown: (e) => {
      if (!answers(e)) return;
      const menu = menuKey(e);
      if (menu != null) {
        e.preventDefault();
        if (menu === "F10" && !e.repeat) openMenu();
        return;
      }
      if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
      const key = Triggers.eventKey(e);
      if (!isCursorKey(key) || (isX(key) && moveX == null)) return;
      e.preventDefault();
      // The cursor repeats a held key at its own pace.
      if (e.repeat) heldRef.current = true;
      else press(key);
    },
    onKeyUp: (e) => {
      if (isCursorKey(Triggers.eventKey(e))) return release();
      if (!answers(e) || menuKey(e) !== "ContextMenu") return;
      e.preventDefault();
      openMenu();
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
    setTarget(key);
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
    if (selection == null) return;
    movedFromRef.current = selection.selected;
    place(key);
  };
  return {
    hover: hovering ? target : undefined,
    setHover,
    setCursor,
    target,
    keyboard,
  };
};
