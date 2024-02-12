// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect } from "react";

import { toArray, type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { useSyncedRef } from "@/hooks/ref";
import { useSelect, type UseSelectProps } from "@/hooks/useSelect";
import { useContext } from "@/list/Context";

export type SelectorProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> = Omit<UseSelectProps<K, E>, "data">;

/**
 * Implements selection behavior for a list.
 *
 * @param props - The props for the List.Selector component. These props are identical
 * to the props for {@link useSelect} hook.
 */
export const Selector = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  value,
  ...props
}: SelectorProps<K, E>): null => {
  const {
    data,
    select: { setOnSelect, setClear, onChange },
  } = useContext<K, E>();

  const { onSelect, clear } = useSelect<K, E>({
    data,
    value,
    ...props,
  } as const as UseSelectProps<K, E>);

  const onSelectRef = useSyncedRef(onSelect);

  const handleSelect = useCallback(
    (key: K) => onSelectRef.current?.(key),
    [onSelectRef],
  );

  useEffect(() => {
    setOnSelect(() => handleSelect);
    setClear(() => clear);
  }, [handleSelect, clear]);

  useEffect(() => {
    onChange(value === null ? [] : toArray(value));
  }, [value]);

  return null;
};
