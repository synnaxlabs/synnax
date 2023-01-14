// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState } from "react";

import { InputControlProps } from "@/core/Input";
import { useKeysHeld } from "@/hooks";
import { KeyedRecord } from "@/util/record";
import { ArrayTransform } from "@/util/transform";

export type SelectedRecord<E extends KeyedRecord<E>> = E & {
  selected?: true;
};

export interface UseSelectMultipleProps<E extends KeyedRecord<E>>
  extends InputControlProps<readonly string[]> {
  data: E[];
  allowMultiple?: boolean;
}

export interface UseSelectMultipleReturn<E extends KeyedRecord<E>> {
  transform: ArrayTransform<E, SelectedRecord<E>>;
  onSelect: (key: string) => void;
  clear: () => void;
}

const shiftKeys = ["Shift"];

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
 *
 * @param props - The props for the hook.
 * @param props.data - The data to select from.
 * @param props.value - The current selection state.
 * @param props.onChange - The callback to invoke when the selection state changes.
 * @param props.allowMultiple - Whether to allow multiple selections.
 *
 */
export const useSelectMultiple = <E extends KeyedRecord<E>>({
  data = [],
  value = [],
  allowMultiple = true,
  onChange,
}: UseSelectMultipleProps<E>): UseSelectMultipleReturn<E> => {
  const [shiftValue, setShiftValue] = useState<string | undefined>(undefined);
  const { any: shift } = useKeysHeld(shiftKeys);

  useEffect(() => {
    if (!shift) setShiftValue(undefined);
  }, [shift]);

  const handleChange = useCallback(
    (key: string): void => {
      let nextSelected: readonly string[] = [];
      if (!allowMultiple) {
        nextSelected = value.includes(key) ? [] : [key];
      } else if (shift && shiftValue !== undefined) {
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
        setShiftValue(undefined);
      } else {
        if (shift) setShiftValue(key);
        if (value.includes(key)) nextSelected = value.filter((k) => k !== key);
        else nextSelected = [...value, key];
      }
      nextSelected = [...new Set(nextSelected)];
      onChange(nextSelected);
    },
    [onChange, value, data, shift, shiftValue, allowMultiple]
  );

  const clear = useCallback((): void => onChange([]), [onChange]);

  const transform = useCallback(
    (data: E[]): Array<SelectedRecord<E>> =>
      data.map((d) => (value.includes(d.key) ? { ...d, selected: true } : d)),
    [value]
  );

  return { onSelect: handleChange, clear, transform };
};
