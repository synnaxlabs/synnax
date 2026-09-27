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
  type destructor,
  type location,
  type optional,
  type record,
  unique,
} from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { Dialog } from "@/dialog";
import { useSyncedRef } from "@/hooks/ref";
import { List } from "@/list";
import { useRegistryContext } from "@/select/registry";
import { useHover, type UseHoverProps, type UseHoverReturn } from "@/select/useHover";
import { Triggers } from "@/triggers";

/** Extra information passed as a second argument to a selection's `onChange`. */
export interface UseOnChangeExtra<K extends record.Key = record.Key> {
  /** The key of the entry that was last clicked. */
  clicked: K | null;
}

/** Props for {@link useSingle} when clicking the selected entry can clear it. */
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

type UseSingleInternalProps<K extends record.Key> =
  UseSingleAllowNoneProps<K> | UseSingleRequiredProps<K>;

/** Props for {@link useSingle}. */
export type UseSingleProps<K extends record.Key> = optional.Optional<
  UseSingleInternalProps<K>,
  "allowNone"
> &
  Pick<UseHoverProps<K>, "initialHover" | "enableTriggers">;

/** Props for {@link useMultiple}. */
export interface UseMultipleProps<K extends record.Key> extends Pick<
  UseHoverProps<K>,
  "initialHover" | "enableTriggers"
> {
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
export interface UseReturn<K extends record.Key> extends UseHoverReturn<K> {
  /** Applies a click on the given key, honoring the held modifier keys. */
  onSelect: (key: K) => void;
  /** Replaces the selection outright, ignoring modifiers. */
  setSelected: (keys: K[]) => void;
  /** Empties the selection. A no-op when none is not allowed. */
  clear: () => void;
}

interface Order<K extends record.Key> {
  /** Changes identity whenever the data changes. */
  getOrder: () => K[];
  scrollTo: (key: K, direction: location.Y) => void;
  /** Calls the listener whenever the fixed items change. */
  subscribe: (listener: () => void) => destructor.Destructor;
}

// The registry holds keys of every selection's type, so the order narrows them to K.
const useOrder = <K extends record.Key>(): Order<K> => {
  const registry = useRegistryContext("Select.Frame");
  const { data } = List.useData<K>();
  const { scrollToIndex } = List.useScroller();
  const getOrder = useCallback(() => registry.getOrder(data) as K[], [registry, data]);
  const scrollTo = useCallback(
    (key: K, direction: location.Y) => {
      const element = registry.getElement(key);
      if (element != null) element.scrollIntoView({ block: "nearest" });
      else scrollToIndex(data.indexOf(key), direction);
    },
    [registry, data, scrollToIndex],
  );
  return { getOrder, scrollTo, subscribe: registry.subscribe };
};

/**
 * Drives a single-entry selection over the options of the enclosing frame, adding
 * keyboard hover and, when allowed, clear-on-reclick.
 */
export const useSingle = <K extends record.Key>({
  allowNone = false,
  onChange,
  value,
  closeDialogOnSelect = false,
  initialHover,
  enableTriggers,
  autoSelectOnNone = false,
}: UseSingleProps<K>): UseReturn<K> => {
  const valueRef = useSyncedRef(value);
  const { close } = Dialog.useContext();
  const { getOrder, scrollTo, subscribe } = useOrder<K>();
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
  const handleSelect = useCallback(
    (key: K): void => {
      if (valueRef.current === key) {
        if (allowNone) onChange(null as unknown as K, { clicked: null });
        if (closeDialogOnSelect) close();
        return;
      }
      onChange(key, { clicked: key });
      if (closeDialogOnSelect) close();
    },
    [onChange, close],
  );
  const clear = useCallback(() => {
    if (allowNone) onChange(null as unknown as K, { clicked: null });
  }, [onChange, allowNone]);

  const setSelected = useCallback(
    (keys: K[]): void => onChange(keys[0], { clicked: null }),
    [onChange],
  );

  const hover = useHover({
    getOrder,
    subscribe,
    scrollTo,
    onSelect: handleSelect,
    initialHover,
    enableTriggers,
  });
  return { onSelect: handleSelect, setSelected, clear, ...hover };
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
  enableTriggers,
  allowNone = true,
  closeDialogOnSelect = false,
  autoSelectOnNone = false,
}: UseMultipleProps<K>): UseReturn<K> => {
  const shiftValueRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });
  const ctrl = Triggers.useHeldRef({ triggers: [["Control"]], loose: true });
  const { close } = Dialog.useContext();
  const valueRef = useSyncedRef(value);
  const { getOrder, scrollTo, subscribe } = useOrder<K>();
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
  const onSelect = useCallback(
    (key: K): void => {
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
    },
    [valueRef, getOrder, onChange, close],
  );
  const clear = useCallback((): void => onChange([], { clicked: null }), [onChange]);
  const setSelected = useCallback(
    (keys: K[]): void => onChange(keys, { clicked: null }),
    [onChange],
  );
  const hover = useHover({
    getOrder,
    subscribe,
    scrollTo,
    onSelect,
    initialHover,
    enableTriggers,
  });
  return { onSelect, setSelected, clear, ...hover };
};
