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

import { createFilterTransform } from "@/hooks";
import { useDebouncedCallback } from "@/hooks/useDebouncedCallback";
import { Input, InputControl } from "@/input";
import { useContext } from "@/list/Context";
import { RenderProp } from "@/util/renderProp";

import { OptionalControl } from "../input/types";

export interface FilterProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends OptionalControl<string> {
  children?: RenderProp<InputControl<string>>;
  debounce?: number;
}

/**
 * Implements in-browser filtration for a list.
 *
 * @param props - The props for the List.Search component.
 * @param props.children - A custom input render prop for the search functionality. This
 * must implement the InputControl<string> interface.
 * @param opts - Custom options for the search functionality. See the {@link fuse.IFuseOptions}
 * interface for more details.
 */
export const Filter = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  children = (props) => <Input {...props} />,
  debounce = 250,
  onChange,
  value,
}: FilterProps<K, E>): ReactElement | null => {
  const [internalValue, setInternalValue] = Input.usePassthrough<string>({
    onChange,
    value,
    initialValue: "",
  });
  const { setTransform, deleteTransform } = useContext<K, E>();

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
