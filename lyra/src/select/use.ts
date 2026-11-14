// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  type location,
  type optional,
  type record,
  unique,
} from "@synnaxlabs/x";
import { type KeyboardEvent, useEffect, useRef } from "react";

import { Dialog } from "@/dialog";
import { useSyncedRef } from "@/hooks/ref";
import { List } from "@/list";
import { useRegistryContext } from "@/select/registry";
import {
  type Cursor,
  type SelectionOrder,
  useCursor,
  type UseCursorProps,
  type UseCursorReturn,
} from "@/select/useCursor";
import { Triggers } from "@/triggers";

/** Extra information passed as a second argument to a selection's `onChange`. */
export interface UseOnChangeExtra<K extends record.Key = record.Key> {
  /** The key of the entry that was last clicked. */
  clicked: K | null;
}

/**
 * Props for {@link useSingle} when the selection can be cleared. Clicking the selected
 * entry clears it only when the click does not close a dialog.
 */
export interface UseSingleAllowNoneProps<K extends record.Key> {
  value?: K;
  onChange: (next: K | null, extra: UseOnChangeExtra<K>) => void;
  allowNone?: true;
  /** Whether to close the enclosing dialog after a selection. */
  closeDialogOnSelect?: boolean;
  /** Whether to select the first option whenever the value names none of them. */
  autoSelectOnNone?: boolean;
}

/** Props for {@link useSingle} when a selection is mandatory. */
export interface UseSingleRequiredProps<K extends record.Key> {
  value: K;
  onChange: (next: K, extra: UseOnChangeExtra<K>) => void;
  allowNone: false | undefined;
  closeDialogOnSelect?: boolean;
  autoSelectOnNone?: boolean;
}

type CursorProps<K extends record.Key> = Pick<
  UseCursorProps<K>,
  "initialHover" | "enableTriggers" | "moveX"
> & { cursor?: Cursor };

type UseSingleInternalProps<K extends record.Key> =
  UseSingleAllowNoneProps<K> | UseSingleRequiredProps<K>;

/** Props for {@link useSingle}. */
export type UseSingleProps<K extends record.Key> = optional.Optional<
  UseSingleInternalProps<K>,
  "allowNone"
> &
  CursorProps<K>;

/** Props for {@link useMultiple}. */
export interface UseMultipleProps<K extends record.Key> extends CursorProps<K> {
  /** Whether the user can deselect the last remaining entry. Defaults to true. */
  allowNone?: boolean;
  value: K[];
  onChange: (next: K[], extra: UseOnChangeExtra<K>) => void;
  /**
   * Whether an unmodified click replaces the selection instead of adding to it. Shift
   * and control still extend and toggle.
   */
  replaceOnSingle?: boolean;
  /** Whether to close the enclosing dialog after a selection. */
  closeDialogOnSelect?: boolean;
  /** Whether to select the first option whenever the value names none of them. */
  autoSelectOnNone?: boolean;
}

/**
 * hasModifier reports whether a pointer event carries a selection modifier, meaning
 * the gesture is aimed at the selection rather than at the item: shift extends a
 * range from the anchor, control (or command) toggles a single key. A row that
 * activates on click routes to the enclosing selection instead when this is true.
 */
export const hasModifier = (e: {
  shiftKey: boolean;
  ctrlKey: boolean;
  metaKey: boolean;
}): boolean => e.shiftKey || e.ctrlKey || e.metaKey;

/** Return value for the {@link useSingle} and {@link useMultiple} hooks. */
export interface UseReturn<K extends record.Key> extends Omit<
  UseCursorReturn<K>,
  "setCursor" | "target"
> {
  /** Applies a click on the given key, honoring the held modifier keys. */
  onSelect: (key: K) => void;
  /** Replaces the selection outright, ignoring modifiers. */
  setSelected: (keys: K[]) => void;
  /** Empties the selection. A no-op when none is not allowed. */
  clear: () => void;
}

// The registry holds keys of every selection's type, so the order narrows them to K.
const useOrder = <K extends record.Key>(): SelectionOrder<K> => {
  const registry = useRegistryContext("Select.Frame");
  const { scrollToIndex, getData, subscribeData } = List.useUtilContext<K>();
  const getRowElement = List.useGetElement();
  const getOrder = () => registry.getOrder(getData()) as K[];
  const getElement = (key: K) =>
    registry.hasItem(key) ? registry.getElement(key) : getRowElement(key);
  const scrollTo = (key: K, direction: location.Y) => {
    const element = registry.getElement(key);
    if (element != null) element.scrollIntoView({ block: "nearest" });
    else scrollToIndex(getData().indexOf(key), direction);
  };
  const click = (key: K) => getElement(key)?.click();
  const doubleClick = (key: K) =>
    getElement(key)?.dispatchEvent(
      new MouseEvent("dblclick", { bubbles: true, cancelable: true }),
    );
  const contextMenu = (key: K) => {
    const element = getElement(key);
    if (element == null) return;
    const { left, bottom } = element.getBoundingClientRect();
    element.dispatchEvent(
      new MouseEvent("contextmenu", {
        bubbles: true,
        cancelable: true,
        clientX: left,
        clientY: bottom,
      }),
    );
  };
  return {
    getOrder,
    scrollTo,
    click,
    doubleClick,
    contextMenu,
    subscribe: (listener) => {
      const stopRegistry = registry.subscribe(listener);
      const stopData = subscribeData(listener);
      return () => {
        stopRegistry();
        stopData();
      };
    },
  };
};

/**
 * Drives a single-entry selection over the options of the enclosing frame, adding a
 * keyboard cursor and, when allowed, clear-on-reclick.
 */
export const useSingle = <K extends record.Key>({
  allowNone = false,
  onChange,
  value,
  closeDialogOnSelect = false,
  initialHover,
  cursor,
  enableTriggers,
  moveX,
  autoSelectOnNone = false,
}: UseSingleProps<K>): UseReturn<K> => {
  const valueRef = useSyncedRef(value);
  const { close } = Dialog.useContext();
  const order = useOrder<K>();
  const { getOrder, subscribe } = order;
  const setSelected = (keys: K[]): void => onChange(keys[0], { clicked: null });
  const { setCursor, target, ...rest } = useCursor({
    order,
    selected: value == null ? [] : [value],
    select: (key) => setSelected([key]),
    initialHover,
    cursor,
    enableTriggers,
    moveX,
  });
  useEffect(() => {
    if (!autoSelectOnNone) return;
    const select = (): void => {
      const order = getOrder();
      if (order.length > 0 && (value == null || !order.includes(value)))
        onChange(order[0], { clicked: order[0] });
    };
    select();
    return subscribe(select);
  }, [autoSelectOnNone, onChange, value, getOrder, subscribe]);
  const clear = () => {
    if (allowNone) onChange(null as unknown as K, { clicked: null });
  };
  const handleSelect = (key: K): void => {
    setCursor(key);
    if (valueRef.current === key) {
      if (closeDialogOnSelect) close();
      else clear();
      return;
    }
    onChange(key, { clicked: key });
    if (closeDialogOnSelect) close();
  };
  List.usePin(target);
  return { onSelect: handleSelect, setSelected, clear, ...rest };
};

/**
 * Drives a multi-entry selection over the options of the enclosing frame. Shift extends
 * a range from the last click, control toggles one key, and a plain click adds or
 * removes unless `replaceOnSingle` is set.
 */
export const useMultiple = <K extends record.Key>({
  value = [],
  replaceOnSingle = false,
  onChange,
  initialHover,
  cursor,
  enableTriggers,
  moveX,
  allowNone = true,
  closeDialogOnSelect = false,
  autoSelectOnNone = false,
}: UseMultipleProps<K>): UseReturn<K> => {
  const shiftValueRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });
  const ctrl = Triggers.useHeldRef({ triggers: [["Control"]], loose: true });
  const { close } = Dialog.useContext();
  const valueRef = useSyncedRef(value);
  const order = useOrder<K>();
  const { getOrder, subscribe } = order;
  // A keyboard move selects as a plain click does, so it also anchors the next shift.
  const select = (key: K): void => {
    shiftValueRef.current = key;
    onChange([key], { clicked: null });
  };
  const { setCursor, target, ...rest } = useCursor({
    order,
    selected: value,
    select,
    initialHover,
    cursor,
    enableTriggers,
    moveX,
  });
  useEffect(() => {
    if (!autoSelectOnNone) return;
    const select = (): void => {
      const order = getOrder();
      if (order.length > 0 && !order.some((k) => value.includes(k)))
        onChange([order[0]], { clicked: order[0] });
    };
    select();
    return subscribe(select);
  }, [autoSelectOnNone, onChange, value, getOrder, subscribe]);
  const onSelect = (key: K): void => {
    setCursor(key);
    const shiftValue = shiftValueRef.current;
    let nextSelected: K[];
    const value = array.toArray(valueRef.current).filter((v) => v != null);
    // If the control key is held, we can still allow multiple selection.
    if (ctrl.current.held && replaceOnSingle)
      if (value.includes(key)) nextSelected = value.filter((k) => k !== key);
      else nextSelected = [...value, key];
    else if (shift.current.held && shiftValue !== null) {
      const order = getOrder();
      // We might select in reverse order, so we need to sort the indexes.
      const [start, end] = [order.indexOf(key), order.indexOf(shiftValue)].sort(
        (a, b) => a - b,
      );
      const nextKeys = order.slice(start, end + 1);
      // We already deselect the shiftSelected key, so we don't included it
      // when checking whether to select or deselect the entire range.
      if (
        nextKeys.slice(1, nextKeys.length - 1).every((k) => value.includes(k)) &&
        value.includes(key)
      )
        nextSelected = value.filter((k) => !nextKeys.includes(k));
      else nextSelected = [...value, ...nextKeys];
      shiftValueRef.current = null;
    } else {
      shiftValueRef.current = key;
      if (replaceOnSingle)
        nextSelected = value.includes(key) && value.length === 1 ? [] : [key];
      else if (value.includes(key)) nextSelected = value.filter((k) => k !== key);
      else nextSelected = [...value, key];
    }
    const v = unique.unique(nextSelected);
    if (v.length === 0) {
      if (!allowNone) return;
      shiftValueRef.current = null;
    }
    onChange(v, { clicked: key });
    if (closeDialogOnSelect) close();
  };
  const clear = (): void => onChange([], { clicked: null });
  const setSelected = (keys: K[]): void => onChange(keys, { clicked: null });
  List.usePin(target);
  return { onSelect, setSelected, clear, ...rest };
};

/**
 * Stops Enter from clicking a focused trigger while its options are open, as
 * {@link useHover} gives that Enter to the hovered option.
 */
export const preventEnterWhileOpen = (e: KeyboardEvent, visible: boolean): void => {
  if (visible && e.key === "Enter") e.preventDefault();
};
