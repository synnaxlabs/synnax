// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useEffect, useRef } from "react";

import {
  type AsyncTermSearcher,
  type Key,
  type KeyedRenderableRecord,
} from "@synnaxlabs/x";

import { useSyncedRef } from "@/hooks";
import { useDebouncedCallback } from "@/hooks/useDebouncedCallback";
import { Input } from "@/input";
import { useContext } from "@/list/Context";
import { state } from "@/state";
import { Status } from "@/status";
import { type RenderProp, componentRenderProp } from "@/util/renderProp";

export interface SearchProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Input.OptionalControl<string> {
  searcher?: AsyncTermSearcher<string, K, E>;
  debounce?: number;
  children?: RenderProp<Input.Control<string>>;
  pageSize?: number;
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
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  debounce = 250,
  children = componentRenderProp(Input.Text),
  searcher,
  value,
  onChange,
  pageSize = 10,
}: SearchProps<K, E>): ReactElement | null => {
  const [internalValue, setInternvalValue] = state.usePurePassthrough({
    value,
    onChange,
    initial: "",
  });
  const valueRef = useSyncedRef(internalValue);
  const promiseOut = useRef<boolean>(false);
  const hasMore = useRef(true);
  const offset = useRef(0);
  const {
    setSourceData,
    setEmptyContent,
    infinite: { setOnFetchMore, setHasMore },
  } = useContext<K, E>();
  useEffect(() => setEmptyContent(NO_TERM), [setEmptyContent]);

  const handleFetchMore = useCallback(
    (reset: boolean = false) => {
      if (valueRef.current.length > 0 || promiseOut.current || searcher == null) return;
      if (reset) {
        offset.current = 0;
        hasMore.current = true;
        setHasMore(true);
      }
      promiseOut.current = true;
      searcher
        .page(offset.current, pageSize)
        .then((r) => {
          promiseOut.current = false;
          if (r.length < pageSize) {
            hasMore.current = false;
            setHasMore(false);
          }
          offset.current += pageSize;
          if (reset) setSourceData(r);
          else setSourceData((d) => [...d, ...r]);
        })
        .catch((e) => {
          promiseOut.current = false;
          console.error(e);
        });
    },
    [searcher, setSourceData, pageSize],
  );

  useEffect(() => {
    console.log("B");
    handleFetchMore(true);
    setOnFetchMore(handleFetchMore);
  }, [handleFetchMore]);

  const debounced = useDebouncedCallback(
    (term: string) => {
      if (term.length === 0) {
        console.log("CD");
        handleFetchMore(true);
        return;
      }
      if (searcher == null) return setEmptyContent(NO_RESULTS);
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
            </Status.Text.Centered>,
          );
        });
    },
    debounce,
    [searcher, setSourceData, setEmptyContent],
  );

  const handleChange = useCallback(
    (term: string) => {
      setInternvalValue(term);
      debounced(term);
    },
    [setInternvalValue, debounced],
  );

  return children({ value: internalValue, onChange: handleChange });
};
