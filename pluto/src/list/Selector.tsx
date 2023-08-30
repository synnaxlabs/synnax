// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useSelectMultiple, UseSelectMultipleProps } from "@/hooks/useSelectMultiple";
import { useContext } from "@/list/Context";

export interface SelectorProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<UseSelectMultipleProps<K, E>, "data"> {}

/**
 * Implements selection behavior for a list.
 *
 * @param props - The props for the List.Selector component. These props are identical
 * to the props for {@link useSelectMultiple} hook.
 */
export const Selector = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  value,
  ...props
}: SelectorProps<K, E>): null => {
  const {
    data,
    select: { setOnSelect, setClear, onChange },
  } = useContext<K, E>();

  const { onSelect, clear } = useSelectMultiple({
    data,
    value,
    ...props,
  });

  useEffect(() => {
    setOnSelect(() => onSelect);
    setClear(() => clear);
  }, [onSelect, clear]);

  useEffect(() => {
    onChange(value);
  }, [value]);

  return null;
};
