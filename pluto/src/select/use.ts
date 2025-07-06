// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type Optional, type record, unique } from "@synnaxlabs/x";
import { useCallback, useRef } from "react";

import { useSyncedRef } from "@/hooks/ref";
import { Triggers } from "@/triggers";

/**
 * Extra information passed as an additional argument to the `onChange` callback.
 * of the {@link useMultiple} hook.
 */
export interface UseOnChangeExtra<K extends record.Key = record.Key> {
  clickedIndex: number | null;
  /** The key of the entry that was last clicked. */
  clicked: K | null;
}

interface BaseProps<K extends record.Key> {
  data: K[];
  replaceOnSingle?: boolean;
}

export interface UseSingleAllowNoneProps<K extends record.Key> extends BaseProps<K> {
  allowNone?: true;
  value: K | null;
  onChange: (next: K | null, extra: UseOnChangeExtra<K>) => void;
}

export interface UseSingleRequiredProps<K extends record.Key> extends BaseProps<K> {
  allowNone: false | undefined;
  value: K;
  onChange: (next: K, extra: UseOnChangeExtra<K>) => void;
}

type UseSingleInternalProps<K extends record.Key> =
  | UseSingleAllowNoneProps<K>
  | UseSingleRequiredProps<K>;

export type UseSingleProps<K extends record.Key> = Optional<
  UseSingleInternalProps<K>,
  "allowNone"
>;

export interface UseMultipleProps<K extends record.Key> extends BaseProps<K> {
  allowMultiple?: true;
  value: K | K[];
  onChange: (next: K[], extra: UseOnChangeExtra<K>) => void;
}

/** Return value for the {@link useMultiple} hook. */
export interface UseReturn<K extends record.Key = record.Key> {
  onSelect: (key: K) => void;
  clear: () => void;
}

export const selectValueIsZero = <K extends record.Key>(
  value: K | K[] | null,
): value is null | K[] => {
  if (value == null) return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "string") return value.length === 0;
  return false;
};

export const useSingle = <K extends record.Key>({
  data = [],
  allowNone = false,
  onChange,
}: UseSingleProps<K>): UseReturn<K> => {
  const dataRef = useSyncedRef(data);
  const handleSelect = useCallback(
    (key: K): void => {
      const clickedIndex = dataRef.current.findIndex((v) => v === key);
      onChange(key, { clicked: key, clickedIndex });
    },
    [dataRef, onChange],
  );
  const clear = useCallback(() => {
    if (allowNone)
      onChange(null as unknown as K, { clicked: null, clickedIndex: null });
  }, [onChange, allowNone]);
  return { onSelect: handleSelect, clear };
};

export const useMultiple = <K extends record.Key>({
  data = [],
  value = [],
  replaceOnSingle = false,
  onChange,
}: UseMultipleProps<K>): UseReturn<K> => {
  const shiftValueRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });
  const ctrl = Triggers.useHeldRef({ triggers: [["Control"]], loose: true });
  const valueRef = useSyncedRef(value);
  const dataRef = useSyncedRef(data);
  const onSelect = useCallback(
    (key: K): void => {
      const shiftValue = shiftValueRef.current;
      const data = dataRef.current;
      let nextSelected: K[] = [];
      const value = array.toArray(valueRef.current).filter((v) => v != null);
      // If the control key is held, we can still allow multiple selection.
      if (ctrl.current.held && replaceOnSingle)
        if (value.includes(key))
          // Remove the key if it's already selected.
          nextSelected = value.filter((k) => k !== key);
        // Add it if its not.
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
      if (v.length === 0) shiftValueRef.current = null;
      onChange(v, {
        clicked: key,
        clickedIndex: data.findIndex((v) => v === key),
      });
    },
    [valueRef, dataRef, onChange],
  );
  const clear = useCallback(
    (): void => onChange([], { clicked: null, clickedIndex: 0 }),
    [onChange],
  );
  return { onSelect, clear };
};
