// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Keyed, type Optional, toArray, unique } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { useSyncedRef } from "@/hooks/ref";
import { Triggers } from "@/triggers";

/**
 * Extra information passed as an additional argument to the `onChange` callback.
 * of the {@link useSelect} hook.
 */
export interface UseSelectOnChangeExtra<
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
> {
  clickedIndex: number | null;
  /** The key of the entry that was last clicked. */
  clicked: K | null;
  /** The entries that are currently selected. */
  entries: E[];
}

interface BaseProps<K extends Key, E extends Keyed<K>> {
  data?: E[] | (() => E[]);
  replaceOnSingle?: boolean;
}

export interface UseSelectSingleAllowNoneProps<K extends Key, E extends Keyed<K>>
  extends BaseProps<K, E> {
  allowMultiple: false;
  allowNone?: true;
  autoSelectOnNone?: boolean;
  value: K | null;
  onChange: (next: K | null, extra: UseSelectOnChangeExtra<K, E>) => void;
}

export interface UseSelectSingleDisallowNoneProps<K extends Key, E extends Keyed<K>>
  extends BaseProps<K, E> {
  allowMultiple: false;
  allowNone: false | undefined;
  autoSelectOnNone?: boolean;
  value: K;
  onChange: (next: K, extra: UseSelectOnChangeExtra<K, any>) => void;
}

type UseSelectSingleInternalProps<K extends Key, E extends Keyed<K>> =
  | UseSelectSingleAllowNoneProps<K, E>
  | UseSelectSingleDisallowNoneProps<K, E>;

export type UseSelectSingleProps<K extends Key, E extends Keyed<K>> = Optional<
  UseSelectSingleInternalProps<K, E>,
  "allowNone"
>;

export interface UseSelectMultipleProps<K extends Key, E extends Keyed<K>>
  extends BaseProps<K, E> {
  allowMultiple?: true;
  allowNone?: boolean;
  autoSelectOnNone?: boolean;
  value: K | K[];
  onChange: (next: K[], extra: UseSelectOnChangeExtra<K, E>) => void;
}

/** Props for the {@link useSelect} hook. */
export type UseSelectProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>> =
  | UseSelectSingleProps<K, E>
  | UseSelectMultipleProps<K, E>;

export type FlexUseSelectProps<K extends Key, E extends Keyed<K>> = {
  data?: E[] | (() => E[]);
  value: K | K[] | null;
  allowMultiple?: boolean;
  allowNone?: boolean;
  autoSelectOnNone?: boolean;
  replaceOnSingle?: boolean;
  onChange: (next: K | K[] | null, extra: UseSelectOnChangeExtra<K, E>) => void;
};

/** Return value for the {@link useSelect} hook. */
export interface UseSelectMultipleReturn<K extends Key = Key> {
  onSelect: (key: K) => void;
  clear: () => void;
}

export const selectValueIsZero = <K extends Key>(
  value: K | K[] | null,
): value is null | K[] => {
  if (value == null) return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "string") return value.length === 0;
  return false;
};

const DEFAULT_PROPS_DATA: any[] = [];
const DEFAULT_PROPS_VALUE: any[] = [];

/**
 * Implements generic selection over a collection of keyed records. The hook
 * does not maintain internal selection state, but instead relies on the `value` and
 * `onChange` props to manage the selection state. This allows the hook to be used
 * with any selection state implementation, such as a React state hook or a Redux
 * store.
 *
 * The hook also supports shift-selection of a range. This means that the data passed in
 * must be in the same order/cardinality as the data that is displayed.
 *
 * It's important to note that the hook implements the InputControl interface, which
 *  means that it can be used as a controlled input in a form.
 *
 * @param props - The props for the hook.
 * @param props.data - The data to select from.
 * @param props.value - The current selection state.
 * @param props.onChange - The callback to invoke when the selection state changes.
 * @param props.allowMultiple - Whether to allow multiple selections.
 *
 * @returns transform - A transform that can be used to add a `selected` property to
 * each record in the data.
 * @returns onSelect - A callback that can be used to select a record. This should
 * probably be passed to the `onClick` corresponding to each record.
 * @returns clear - A callback that can be used to clear the selection.
 */
export const useSelect = <K extends Key, E extends Keyed<K>>({
  data: propsData = DEFAULT_PROPS_DATA,
  value: propsValue = DEFAULT_PROPS_VALUE,
  allowMultiple,
  allowNone,
  replaceOnSingle = false,
  autoSelectOnNone = false,
  onChange,
}: UseSelectProps<K, E> | FlexUseSelectProps<K, E>): UseSelectMultipleReturn<K> => {
  const shiftValueRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });
  const ctrl = Triggers.useHeldRef({ triggers: [["Control"]], loose: true });

  const valueRef = useSyncedRef(propsValue);
  const dataRef = useSyncedRef(propsData);

  const handleChange = useCallback(
    (next: K[], extra: UseSelectOnChangeExtra<K, E>) => {
      valueRef.current = next;
      if (next.length === 0 && allowNone !== false) {
        if (allowMultiple !== false) return onChange([], extra);
        return onChange(null as unknown as K, extra);
      }
      if (allowMultiple !== false) return onChange(next, extra);
      if (next.length > 0) return onChange(next[0], extra);
    },
    [onChange, allowNone, allowMultiple],
  );

  useEffect(() => {
    let data = dataRef.current;
    if (!Array.isArray(data)) data = data();
    // If for some reason the value is empty and it shouldn't be, automatically set
    // it to the new value..
    if (
      selectValueIsZero(propsValue) &&
      allowNone === false &&
      data.length > 0 &&
      autoSelectOnNone
    ) {
      const first = data[0];
      shiftValueRef.current = first.key;
      handleChange([first.key], {
        entries: [first],
        clicked: first.key,
        clickedIndex: 0,
      });
    }
  }, [handleChange, dataRef, propsValue, allowNone]);

  const onSelect = useCallback(
    (key: K): void => {
      const shiftValue = shiftValueRef.current;
      let data = dataRef.current;
      if (!Array.isArray(data)) data = data();
      let nextSelected: K[] = [];
      const value = toArray(valueRef.current).filter((v) => v != null) as K[];
      // Simple case. If we can't allow multiple, then just toggle the key.
      if (allowMultiple === false) nextSelected = value.includes(key) ? [] : [key];
      // If the control key is held, we can still allow multiple selection.
      else if (ctrl.current.held && replaceOnSingle)
        if (value.includes(key))
          // Remove the key if it's already selected.
          nextSelected = value.filter((k) => k !== key);
        // Add it if its not.
        else nextSelected = [...value, key];
      else if (shift.current.held && shiftValue !== null) {
        // We might select in reverse order, so we need to sort the indexes.
        const [start, end] = [
          data.findIndex((v) => v.key === key),
          data.findIndex((v) => v.key === shiftValue),
        ].sort((a, b) => a - b);
        const nextKeys = data.slice(start, end + 1).map(({ key }) => key);
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
      const v = unique(nextSelected);
      if (allowNone === false && v.length === 0)
        // If we're not allowed to have no select, still call handleChange with the same
        // value. This is useful when you want to close a dialog on selection.
        return handleChange(value, {
          entries: data.filter(({ key }) => value.includes(key)),
          clicked: key,
          clickedIndex: data.findIndex(({ key: k }) => k === key),
        });
      if (v.length === 0) shiftValueRef.current = null;
      handleChange(v, {
        entries: data.filter(({ key }) => nextSelected.includes(key)),
        clicked: key,
        clickedIndex: data.findIndex(({ key: k }) => k === key),
      });
    },
    [valueRef, dataRef, handleChange, allowMultiple, allowNone],
  );

  const clear = useCallback(
    (): void => handleChange([], { entries: [], clicked: null, clickedIndex: 0 }),
    [handleChange],
  );

  return { onSelect, clear };
};
