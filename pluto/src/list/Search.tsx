// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type AsyncTermSearcher, type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";

import { useSyncedRef } from "@/hooks";
import { useDebouncedCallback } from "@/hooks/useDebouncedCallback";
import { type Input } from "@/input";
import { Text as InputText } from "@/input/Text";
import { useDataUtils } from "@/list/Data";
import { useInfiniteUtils } from "@/list/Infinite";
import { state } from "@/state";
import { Status } from "@/status";
import { type RenderProp } from "@/util/renderProp";

export interface UseSearchProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends Input.OptionalControl<string> {
  searcher?: AsyncTermSearcher<string, K, E>;
  debounce?: number;
  pageSize?: number;
  filter?: (items: E[]) => E[];
}

export interface SearchProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends UseSearchProps<K, E> {
  children?: RenderProp<Input.Control<string>>;
}

export interface UseSearchReturn extends Input.Control<string> {}

const STYLE = {
  height: 150,
};

const NoResults = (): ReactElement => (
  <Status.Text.Centered level="h4" variant="disabled" hideIcon style={STYLE}>
    No Results
  </Status.Text.Centered>
);

const NoTerm = (): ReactElement => (
  <Status.Text.Centered level="h4" variant="disabled" hideIcon style={STYLE}>
    Type to search
  </Status.Text.Centered>
);

const Loading = (): ReactElement => (
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

const defaultFilter = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>(
  items: E[],
): E[] => items;

export const useSearch = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  debounce = 250,
  searcher,
  value,
  onChange,
  pageSize = 15,
  filter = defaultFilter,
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
  const { setSourceData, setEmptyContent, getDefaultEmptyContent } = useDataUtils<
    K,
    E
  >();
  const { setHasMore, setOnFetchMore } = useInfiniteUtils();

  useEffect(() => setEmptyContent(<NoTerm />), [setEmptyContent]);

  const handleFetchMore = useCallback(
    (reset: boolean = false) => {
      if (valueRef.current.length > 0 || promiseOut.current || searcher == null) return;
      if (reset) {
        offset.current = 0;
        hasMore.current = true;
        setHasMore(true);
      }
      promiseOut.current = true;
      setEmptyContent(<Loading />);
      const fn = async () => {
        try {
          let r = await searcher.page(offset.current, pageSize);
          r = filter(r);
          if (r.length === 0)
            setEmptyContent(getDefaultEmptyContent() ?? <NoResults />);
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
    [searcher, setSourceData, pageSize, filter],
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
      if (searcher == null) return setEmptyContent(<NoResults />);
      searcher
        .search(term)
        .then((d) => {
          if (d.length === 0) setEmptyContent(<NoResults />);
          setSourceData(filter(d));
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
    [searcher, setSourceData, setEmptyContent, filter],
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

const searchInput = (props: Input.Control<string>): ReactElement => (
  <InputText placeholder="Search" {...props} />
);

export const Search = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  children = searchInput,
  ...rest
}: SearchProps<K, E>): ReactElement | null => children(useSearch(rest));
