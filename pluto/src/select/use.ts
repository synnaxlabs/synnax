// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type Optional, type record, unique } from "@synnaxlabs/x";
import { type MouseEvent, useCallback, useEffect, useRef } from "react";

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
  clickedIndex: number | null;
  /** The key of the entry that was last clicked. */
  clicked: K | null;
}

export interface UseSingleAllowNoneProps<K extends record.Key> {
  value?: K;
  onChange: (next: K | null, extra: UseOnChangeExtra<K>) => void;
  allowNone?: true;
  closeDialogOnSelect?: boolean;
  autoSelectOnNone?: boolean;
}

export interface UseSingleRequiredProps<K extends record.Key> {
  value: K;
  onChange: (next: K, extra: UseOnChangeExtra<K>) => void;
  allowNone: false | undefined;
  closeDialogOnSelect?: boolean;
  autoSelectOnNone?: boolean;
}

type UseSingleInternalProps<K extends record.Key> =
  | UseSingleAllowNoneProps<K>
  | UseSingleRequiredProps<K>;

export type UseSingleProps<K extends record.Key> = Optional<
  UseSingleInternalProps<K>,
  "allowNone"
> &
  Pick<UseHoverProps<K>, "initialHover">;

export interface UseMultipleProps<K extends record.Key>
  extends Pick<UseHoverProps<K>, "initialHover"> {
  allowNone?: boolean;
  value: K[];
  onChange: (next: K[], extra: UseOnChangeExtra<K>) => void;
  replaceOnSingle?: boolean;
  closeDialogOnSelect?: boolean;
  autoSelectOnNone?: boolean;
}

interface OnSelectEvent extends Pick<MouseEvent, "button"> {}

/** Return value for the {@link useMultiple} hook. */
export interface UseReturn<K extends record.Key> extends UseHoverReturn<K> {
  onSelect: (key: K, e?: OnSelectEvent) => void;
  setSelected: (keys: K[]) => void;
  clear: () => void;
}

export const useSingle = <K extends record.Key>({
  allowNone = false,
  onChange,
  value,
  closeDialogOnSelect = false,
  initialHover,
  autoSelectOnNone = false,
}: UseSingleProps<K>): UseReturn<K> => {
  const valueRef = useSyncedRef(value);
  const { data } = List.useData<K>();
  const { close } = Dialog.useContext();
  const dataRef = useSyncedRef(data);
  useEffect(() => {
    if (autoSelectOnNone && value == null && data.length > 0)
      onChange(data[0], { clicked: data[0], clickedIndex: 0 });
  }, [autoSelectOnNone, onChange, value, data.length]);
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

  const hover = useHover({ data, onSelect: handleSelect, initialHover });
  return { onSelect: handleSelect, setSelected, clear, ...hover };
};

export const useMultiple = <K extends record.Key>({
  value = [],
  replaceOnSingle = false,
  onChange,
  initialHover,
  allowNone = true,
  closeDialogOnSelect = false,
  autoSelectOnNone = false,
}: UseMultipleProps<K>): UseReturn<K> => {
  const { data } = List.useData<K>();
  const shiftValueRef = useRef<K | null>(null);
  const rightClickRef = useRef<K | null>(null);
  const shift = Triggers.useHeldRef({ triggers: [["Shift"]], loose: true });
  const ctrl = Triggers.useHeldRef({ triggers: [["Control"]], loose: true });
  const valueRef = useSyncedRef(value);
  const dataRef = useSyncedRef(data);
  useEffect(() => {
    if (autoSelectOnNone && value.length === 0 && data.length > 0)
      onChange([data[0]], { clicked: data[0], clickedIndex: 0 });
  }, [autoSelectOnNone, onChange, value, data.length]);
  const onSelect = useCallback(
    (key: K, e?: OnSelectEvent): void => {
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
        const isMouseRight = e?.button == Triggers.MOUSE_RIGHT_NUMBER;
        nextSelected = [...value];

        if (isMouseRight) {
          if (rightClickRef.current != null)
            nextSelected = nextSelected.filter((k) => k !== rightClickRef.current);
          if (!nextSelected.includes(key)) rightClickRef.current = key;
        }

        if (replaceOnSingle && (!isMouseRight || value.length === 1))
          nextSelected =
            nextSelected.includes(key) && nextSelected.length === 1 ? [] : [key];
        else if (nextSelected.includes(key)) {
          if (!isMouseRight) nextSelected = nextSelected.filter((k) => k !== key);
        } else nextSelected = [...nextSelected, key];
      }
      nextSelected = unique.unique(nextSelected);
      if (nextSelected.length === 0) {
        if (!allowNone) return;
        shiftValueRef.current = null;
      }
      onChange(nextSelected, {
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
  const hover = useHover({ data, onSelect, initialHover });
  return { onSelect, setSelected, clear, ...hover };
};
