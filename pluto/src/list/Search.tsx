// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type AsyncTermSearcher, type Key, type Keyed } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";

import { useSyncedRef } from "@/hooks";
import { useDebouncedCallback } from "@/hooks/useDebouncedCallback";
import { Input } from "@/input";
import { useDataUtilContext } from "@/list/Data";
import { useInfiniteUtilContext } from "@/list/Infinite";
import { state } from "@/state";
import { Status } from "@/status";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

export interface UseSearchProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends Input.OptionalControl<string> {
  searcher?: AsyncTermSearcher<string, K, E>;
  debounce?: number;
  pageSize?: number;
}

export interface SearchProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends UseSearchProps<K, E> {
  children?: RenderProp<Input.Control<string>>;
}

export interface UseSearchReturn extends Input.Control<string> {}

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
    Type to search
  </Status.Text.Centered>
);

const LOADING = (
  <Status.Text.Centered level="h2" variant="disabled" hideIcon style={STYLE}>
    <Icon.Loading />
  </Status.Text.Centered>
);

interface ErrorEmptyContentProps {
  error: Error;
}

const ErrorEmptyContent = ({ error }: ErrorEmptyContentProps): ReactElement => (
  <Status.Text.Centered level="h4" variant="error" style={STYLE}>
    {error.message}
  </Status.Text.Centered>
);

export const useSearch = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  debounce = 250,
  searcher,
  value,
  onChange,
  pageSize = 10,
}: UseSearchProps<K, E>): UseSearchReturn => {
  const [internalValue, setInternalValue] = state.usePurePassthrough({
    value,
    onChange,
    initial: "",
  });
  const valueRef = useSyncedRef(internalValue);
  const promiseOut = useRef<boolean>(false);
  const hasMore = useRef(true);
  const offset = useRef(0);
  const { setSourceData, setEmptyContent, getDefaultEmptyContent } = useDataUtilContext<
    K,
    E
  >();
  const { setHasMore, setOnFetchMore } = useInfiniteUtilContext();

  useEffect(() => setEmptyContent(NO_TERM), [setEmptyContent]);

  const handleFetchMore = useCallback(
    (reset: boolean = false) => {
      console.log("FETCH");
      if (valueRef.current.length > 0 || promiseOut.current || searcher == null) return;
      if (reset) {
        offset.current = 0;
        hasMore.current = true;
        setHasMore(true);
      }
      promiseOut.current = true;
      setEmptyContent(LOADING);
      const fn = async () => {
        try {
          console.log("EXEC");
          const r = await searcher.page(offset.current, pageSize);
          console.log(r);
          if (r.length === 0) setEmptyContent(getDefaultEmptyContent() ?? NO_RESULTS);
          if (r.length < pageSize) {
            hasMore.current = false;
            setHasMore(false);
          }
          offset.current += pageSize;
          if (reset) setSourceData(r);
          else setSourceData((d) => [...d, ...r]);
        } catch (e) {
          promiseOut.current = false;
          setEmptyContent(<ErrorEmptyContent error={e as Error} />);
        } finally {
          promiseOut.current = false;
        }
      };
      void fn();
    },
    [searcher, setSourceData, pageSize],
  );

  useEffect(() => {
    handleFetchMore(true);
    setOnFetchMore(handleFetchMore);
  }, [handleFetchMore]);

  const debounced = useDebouncedCallback(
    (term: string) => {
      if (term.length === 0) {
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
      setInternalValue(term);
      debounced(term);
    },
    [setInternalValue, debounced],
  );

  return { value: internalValue, onChange: handleChange };
};

export const Search = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  children = componentRenderProp(Input.Text),
  ...props
}: SearchProps<K, E>): ReactElement | null => children(useSearch(props));
