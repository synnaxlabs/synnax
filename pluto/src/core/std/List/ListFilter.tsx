// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useState } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import {
  createSearchTransform as newSearchTransform,
  UseSearchTransformProps,
} from "@/core/hooks";
import { useDebouncedCallback } from "@/core/hooks/useDebouncedCallback";
import { Input, InputControl } from "@/core/std/Input";
import { useListContext } from "@/core/std/List/ListContext";
import { RenderProp } from "@/util/renderProp";

export interface ListFilterProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<UseSearchTransformProps<E>, "term"> {
  children?: RenderProp<InputControl<string>>;
  debounce?: number;
}

export const ListFilter = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  children = (props) => <Input {...props} />,
  debounce = 250,
}: ListFilterProps<K, E>): ReactElement | null => {
  const [value, setValue] = useState("");
  const { setTransform, deleteTransform } = useListContext<K, E>();

  const debounced = useDebouncedCallback(setTransform, debounce, []);

  const onChange = useCallback(
    (term: string) => {
      setValue(term);
      if (term.length === 0) deleteTransform("filter");
      else debounced("filter", newSearchTransform({ term }));
    },
    [setValue]
  );

  return children({ value, onChange });
};

export interface Searcher<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {
  search: (term: string) => E[];
}
