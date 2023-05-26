// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";

import { KeyedRenderableRecord } from "@synnaxlabs/x";

import { useListContext } from "@/core/List/ListContext";
import { useSelectMultiple, UseSelectMultipleProps } from "@/hooks/useSelectMultiple";

export interface ListSelectorProps<E extends KeyedRenderableRecord<E>>
  extends Omit<UseSelectMultipleProps<E>, "data"> {}

export const ListSelector = <E extends KeyedRenderableRecord<E>>({
  value,
  ...props
}: ListSelectorProps<E>): null => {
  const {
    data,
    setTransform,
    deleteTransform,
    select: { setOnSelect, setClear },
  } = useListContext<E>();

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
