// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { PartialInputControl } from "../Input/types";

import { createFilterTransform } from "@/core/hooks";
import { useDebouncedCallback } from "@/core/hooks/useDebouncedCallback";
import { Input, InputControl } from "@/core/std/Input";
import { useListContext } from "@/core/std/List/ListContext";
import { RenderProp } from "@/util/renderProp";

export interface ListFilterProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends PartialInputControl<string> {
  children?: RenderProp<InputControl<string>>;
  debounce?: number;
}

export const ListFilter = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  children = (props) => <Input {...props} />,
  debounce = 250,
  onChange,
  value,
}: ListFilterProps<K, E>): ReactElement | null => {
  const [internalValue, setInternalValue] = Input.usePassthrough<string>({
    onChange,
    value,
    initialValue: "",
  });
  const { setTransform, deleteTransform } = useListContext<K, E>();

  const debounced = useDebouncedCallback(setTransform, debounce, []);

  const handleChange = useCallback(
    (term: string) => {
      setInternalValue(term);
      if (term.length === 0) deleteTransform("filter");
      else debounced("filter", createFilterTransform({ term }));
    },
    [setInternalValue]
  );

  return children({ value: internalValue, onChange: handleChange });
};

export interface Searcher<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {
  search: (term: string) => E[];
}
