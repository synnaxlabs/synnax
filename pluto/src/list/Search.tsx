// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useEffect } from "react";

import { AsyncTermSearcher, Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useDebouncedCallback } from "@/hooks/useDebouncedCallback";
import { Input } from "@/input";
import { List } from "@/list";
import { state } from "@/state";
import { Status } from "@/status";
import { RenderProp, componentRenderProp } from "@/util/renderProp";

export interface SearchProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Input.OptionalControl<string> {
  searcher: AsyncTermSearcher<string, K, E>;
  debounce?: number;
  children?: RenderProp<Input.Control<string>>;
}

const STYLE = {
  height: 150,
};

const NO_RESULTS = (
  <Status.Text.Centered level="h4" variant="disabled" hideIcon style={STYLE}>
    No Results
  </Status.Text.Centered>
);

const NO_TERM = (
  <Status.Text.Centered level="h4" variant="disabled" hideIcon style={STYLE}>
    Type to search...
  </Status.Text.Centered>
);

export const Search = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  debounce = 250,
  children = componentRenderProp(Input.Text),
  searcher,
  value,
  onChange,
}: SearchProps<K, E>): ReactElement | null => {
  const [internalValue, setInternvalValue] = state.usePurePassthrough({
    value,
    onChange,
    initial: "",
  });
  const { setSourceData, setEmptyContent } = List.useContext<K, E>();
  useEffect(() => setEmptyContent(NO_TERM), [setEmptyContent]);

  const debounced = useDebouncedCallback(
    (term: string) => {
      searcher
        .search(term)
        .then((d) => {
          if (d.length === 0) setEmptyContent(NO_RESULTS);
          setSourceData(d);
        })
        .catch((e) => {
          setEmptyContent(
            <Status.Text.Centered level="h4" variant="error">
              {e.message}
            </Status.Text.Centered>
          );
        });
    },
    debounce,
    [setSourceData, setEmptyContent]
  );

  const handleChange = useCallback(
    (term: string) => {
      setInternvalValue(term);
      if (term.length === 0) setEmptyContent(NO_TERM);
      else debounced(term);
    },
    [setInternvalValue]
  );

  return children({ value: internalValue, onChange: handleChange });
};
