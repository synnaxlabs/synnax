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

import {
  useSelectMultiple,
  UseSelectMultipleProps,
} from "@/core/hooks/useSelectMultiple";
import { useListContext } from "@/core/std/List/ListContext";

export interface ListSelectorProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<UseSelectMultipleProps<K, E>, "data"> {}

export const ListSelector = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  value,
  ...props
}: ListSelectorProps<K, E>): null => {
  const {
    data,
    setTransform,
    deleteTransform,
    select: { setOnSelect, setClear },
  } = useListContext<K, E>();

  const { onSelect, transform, clear } = useSelectMultiple({
    data,
    value,
    ...props,
  });

  useEffect(() => {
    setOnSelect(() => onSelect);
    setClear(() => clear);
  }, [onSelect, clear]);

  useEffect(() => {
    if (value == null || value.length === 0) deleteTransform("select");
    setTransform("select", transform);
  }, [transform]);

  return null;
};
