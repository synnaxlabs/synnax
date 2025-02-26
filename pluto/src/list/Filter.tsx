// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Keyed } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { createFilterTransform } from "@/hooks";
import { useDebouncedCallback } from "@/hooks/useDebouncedCallback";
import { type Input } from "@/input";
import { Text as InputText } from "@/input/Text";
import { type OptionalControl } from "@/input/types";
import { useDataUtils } from "@/list/Data";
import { state } from "@/state";
import { type RenderProp } from "@/util/renderProp";

export interface UseFilterProps extends OptionalControl<string> {
  debounce?: number;
  transformBefore?: (term: string) => string;
}

export interface FilterProps extends UseFilterProps {
  children?: RenderProp<Input.Control<string>>;
}

export const useFilter = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  debounce = 250,
  value,
  onChange,
  transformBefore,
}: UseFilterProps): Input.Control<string> => {
  const [internalValue, setInternalValue] = state.usePurePassthrough<string>({
    onChange,
    value,
    initial: "",
  });
  const { setTransform, deleteTransform } = useDataUtils<K, E>();

  const debounced = useDebouncedCallback(setTransform, debounce, []);

  const handleChange = useCallback(
    (term: string) => {
      setInternalValue(term);
      if (term.length === 0) deleteTransform("filter");
      else {
        if (transformBefore != null) term = transformBefore(term);
        debounced("filter", createFilterTransform({ term }));
      }
    },
    [setInternalValue, transformBefore],
  );

  return { value: internalValue, onChange: handleChange };
};

/**
 * Implements in-browser filtration for a list.
 *
 * @param props - The props for the List.Search component.
 * @param props.children - A custom input render prop for the search functionality. This
 * must implement the InputControl<string> interface.
 * @param opts - Custom options for the search functionality. See the {@link fuse.IFuseOptions}
 * interface for more details.
 */
export const Filter = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  children = (props) => <InputText placeholder="Filter" {...props} />,
  ...rest
}: FilterProps): ReactElement | null => children(useFilter<K, E>(rest));

export interface Searcher<K extends Key = Key, E extends Keyed<K> = Keyed<K>> {
  search: (term: string) => E[];
}
