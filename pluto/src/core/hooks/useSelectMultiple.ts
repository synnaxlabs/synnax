// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef } from "react";

import { Key, KeyedRecord, unique, ArrayTransform } from "@synnaxlabs/x";

import { Triggers } from "@/core/triggers";

export type SelectedRecord<
  K extends Key,
  E extends KeyedRecord<K, E> = KeyedRecord<K>
> = E & {
  selected?: true;
};

/** Props for the {@link useSelectMultiple} hook. */
export interface UseSelectMultipleProps<
  K extends Key = Key,
  E extends KeyedRecord<K, E> = KeyedRecord<K>
> {
  data: E[];
  allowMultiple?: boolean;
  value: readonly K[];
  onChange: (next: readonly K[], entries: E[]) => void;
}

/** Return value for the {@link useSelectMultiple} hook. */
export interface UseSelectMultipleReturn<
  K extends Key = Key,
  E extends KeyedRecord<K, E> = KeyedRecord<K>
> {
  transform: ArrayTransform<E, SelectedRecord<K, E>>;
  onSelect: (key: K) => void;
  clear: () => void;
}

/**
 * Implements generic multiple selection over a collection of keyed records. The hook
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
export const useSelectMultiple = <
  K extends Key = Key,
  E extends KeyedRecord<K, E> = KeyedRecord<K>
>({
  data = [],
  value = [],
  allowMultiple = true,
  onChange,
}: UseSelectMultipleProps<K, E>): UseSelectMultipleReturn<K, E> => {
  const shiftValueRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });

  const handleSelect = useCallback(
    (key: K): void => {
      let nextSelected: readonly K[] = [];
      const shiftValue = shiftValueRef.current;
      if (!allowMultiple) {
        nextSelected = value.includes(key) ? [] : [key];
      } else if (shift.current.held && shiftValue !== null) {
        // We might select in reverse order, so we need to sort the indexes.
        const [start, end] = [
          data.findIndex((v) => v.key === key),
          data.findIndex((v) => v.key === shiftValue),
        ].sort((a, b) => a - b);
        const nextKeys = data.slice(start, end + 1).map(({ key }) => key);
        // We already deselect the shiftSelected key, so we don't included it
        // when checking whether to select or deselect the entire range.
        if (nextKeys.slice(1, nextKeys.length - 1).every((k) => value.includes(k)))
          nextSelected = value.filter((k) => !nextKeys.includes(k));
        else nextSelected = [...value, ...nextKeys];
        shiftValueRef.current = null;
      } else {
        shiftValueRef.current = shift.current.held ? key : null;
        if (value.includes(key)) nextSelected = value.filter((k) => k !== key);
        else nextSelected = [...value, key];
      }
      onChange(
        unique(nextSelected),
        data.filter(({ key }) => nextSelected.includes(key))
      );
    },
    [onChange, value, data, allowMultiple]
  );

  const clear = useCallback((): void => onChange([], []), [onChange]);

  const transform = useCallback(
    (data: E[]): Array<SelectedRecord<K, E>> =>
      data.map((d) => (value.includes(d.key) ? { ...d, selected: true } : d)),
    [value]
  );

  return { onSelect: handleSelect, clear, transform };
};
