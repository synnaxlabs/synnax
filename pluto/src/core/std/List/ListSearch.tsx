// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useEffect, useState } from "react";

import { AsyncTermSearcher, Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { Input, InputControl } from "../Input";
import { Status } from "../Status";

import { useListContext } from "./ListContext";

import { useDebouncedCallback } from "@/core/hooks/useDebouncedCallback";
import { RenderProp, componentRenderProp } from "@/util/renderProp";

export interface ListSearchProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {
  searcher: AsyncTermSearcher<string, K, E>;
  debounce?: number;
  children?: RenderProp<InputControl<string>>;
  onChange?: (data: E[]) => void;
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

export const ListSearch = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  debounce = 250,
  children = componentRenderProp(Input),
  onChange,
  searcher,
}: ListSearchProps<K, E>): ReactElement | null => {
  const [value, setValue] = useState("");
  const { setSourceData, setEmptyContent } = useListContext<K, E>();
  useEffect(() => setEmptyContent(NO_TERM), [setEmptyContent]);

  const debounced = useDebouncedCallback(
    (term: string) => {
      searcher
        .search(term)
        .then((d) => {
          if (d.length === 0) setEmptyContent(NO_RESULTS);
          setSourceData(d);
          onChange?.(d);
        })
        .catch((e) => {
          setEmptyContent(
            <Status.Text.Centered level="h4" variant="error">
              {e.message}
            </Status.Text.Centered>
          );
          onChange?.([]);
        });
    },
    debounce,
    [setSourceData, setEmptyContent, onChange]
  );

  const handleChange = useCallback(
    (term: string) => {
      setValue(term);
      if (term.length === 0) setEmptyContent(NO_TERM);
      else debounced(term);
    },
    [setValue]
  );

  return children({ value, onChange: handleChange });
};
