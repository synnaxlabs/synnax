// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

import { useStateRef } from "@/hooks";

export interface InfiniteContextValue {
  hasMore: boolean;
  onFetchMore: () => void;
}

export interface InfiniteUtilContextValue {
  setHasMore: (hasMore: boolean) => void;
  setOnFetchMore: (onFetchMore: () => void) => void;
}

const Context = createContext<InfiniteContextValue>({
  hasMore: false,
  onFetchMore: () => undefined,
});

const UtilContext = createContext<InfiniteUtilContextValue>({
  setHasMore: () => undefined,
  setOnFetchMore: () => undefined,
});

export const useInfiniteContext = (): InfiniteContextValue => useContext(Context);

export const useInfiniteUtilContext = (): InfiniteUtilContextValue =>
  useContext(UtilContext);

export interface InfiniteProviderProps extends PropsWithChildren<{}> {}

export const InfiniteProvider = ({ children }: InfiniteProviderProps): ReactElement => {
  const [hasMore, setHasMore] = useState(false);
  const [fetchMoreRef, setOnFetchMore] = useStateRef<(() => void) | undefined>(
    () => undefined,
  );

  const fetchMore = useCallback(() => {
    fetchMoreRef.current?.();
  }, [fetchMoreRef]);

  const ctxValue = useMemo(
    () => ({ hasMore, onFetchMore: fetchMore }),
    [hasMore, fetchMore],
  );

  const utilValue = useMemo<InfiniteUtilContextValue>(
    () => ({
      setHasMore,
      setOnFetchMore: (onFetchMore) => {
        setOnFetchMore(() => onFetchMore);
      },
    }),
    [setHasMore, setOnFetchMore],
  );

  return (
    <Context.Provider value={ctxValue}>
      <UtilContext.Provider value={utilValue}>{children}</UtilContext.Provider>
    </Context.Provider>
  );
};
