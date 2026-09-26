// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type optional, type record, unique } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { Dialog } from "@/dialog";
import { useSyncedRef } from "@/hooks/ref";
import { List } from "@/list";
import { useHover, type UseHoverProps, type UseHoverReturn } from "@/select/useHover";
import { Triggers } from "@/triggers";

/**
 * Extra information passed as an additional argument to the `onChange` callback.
 * of the {@link useMultiple} hook.
 */
export interface UseOnChangeExtra<K extends record.Key = record.Key> {
  /** The index of the clicked entry in the list data. */
  clickedIndex: number | null;
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
  /** Whether to select the first entry whenever the value names nothing in the data. */
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
  /** Whether to select the first entry whenever the value names nothing in the data. */
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

/**
 * Drives a single-entry selection over the enclosing {@link List.Frame}, adding
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
  const { data } = List.useData<K>();
  const { close } = Dialog.useContext();
  const dataRef = useSyncedRef(data);
  useEffect(() => {
    const dataHasValue = value != null && data.includes(value);
    if (autoSelectOnNone && data.length > 0 && !dataHasValue)
      onChange(data[0], { clicked: data[0], clickedIndex: 0 });
  }, [autoSelectOnNone, onChange, value, data.length, data]);
  const handleSelect = useCallback(
    (key: K): void => {
      if (valueRef.current === key) {
        if (allowNone)
          onChange(null as unknown as K, { clicked: null, clickedIndex: null });
        if (closeDialogOnSelect) close();
        return;
      }
      const clickedIndex = dataRef.current.findIndex((v) => v === key);
      onChange(key, { clicked: key, clickedIndex });
      if (closeDialogOnSelect) close();
    },
    [dataRef, onChange, close],
  );
  const clear = useCallback(() => {
    if (allowNone)
      onChange(null as unknown as K, { clicked: null, clickedIndex: null });
  }, [onChange, allowNone]);

  const setSelected = useCallback(
    (keys: K[]): void => onChange(keys[0], { clicked: null, clickedIndex: null }),
    [onChange],
  );

  const hover = useHover({
    data,
    onSelect: handleSelect,
    initialHover,
    enableTriggers,
  });
  return { onSelect: handleSelect, setSelected, clear, ...hover };
};

/**
 * Drives a multi-entry selection over the enclosing {@link List.Frame}. Shift extends a
 * range from the last click, control toggles one key, and a plain click adds or removes
 * unless `replaceOnSingle` is set.
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
  const { data } = List.useData<K>();
  const shiftValueRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });
  const ctrl = Triggers.useHeldRef({ triggers: [["Control"]], loose: true });
  const valueRef = useSyncedRef(value);
  const dataRef = useSyncedRef(data);
  useEffect(() => {
    const dataHasValue = data.some((v) => value.includes(v));
    if (autoSelectOnNone && data.length > 0 && !dataHasValue)
      onChange([data[0]], { clicked: data[0], clickedIndex: 0 });
  }, [autoSelectOnNone, onChange, value, data.length, data]);
  const onSelect = useCallback(
    (key: K): void => {
      const shiftValue = shiftValueRef.current;
      const data = dataRef.current;
      let nextSelected: K[];
      const value = array.toArray(valueRef.current).filter((v) => v != null);
      // If the control key is held, we can still allow multiple selection.
      if (ctrl.current.held && replaceOnSingle)
        if (value.includes(key)) nextSelected = value.filter((k) => k !== key);
        else nextSelected = [...value, key];
      else if (shift.current.held && shiftValue !== null) {
        // We might select in reverse order, so we need to sort the indexes.
        const [start, end] = [
          data.findIndex((v) => v === key),
          data.findIndex((v) => v === shiftValue),
        ].sort((a, b) => a - b);
        const nextKeys = data.slice(start, end + 1).map((v) => v);
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
      onChange(v, {
        clicked: key,
        clickedIndex: data.findIndex((v) => v === key),
      });
      if (closeDialogOnSelect) close();
    },
    [valueRef, dataRef, onChange],
  );
  const clear = useCallback(
    (): void => onChange([], { clicked: null, clickedIndex: 0 }),
    [onChange],
  );
  const setSelected = useCallback(
    (keys: K[]): void => onChange(keys, { clicked: null, clickedIndex: 0 }),
    [onChange],
  );
  const hover = useHover({ data, onSelect, initialHover, enableTriggers });
  return { onSelect, setSelected, clear, ...hover };
};
