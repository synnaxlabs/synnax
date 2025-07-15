// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { type UseListReturn } from "@/flux/list";
import { type Params } from "@/flux/params";

export interface PagerParams extends Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export interface UsePagerReturn {
  onFetchMore: () => void;
  onSearch: (term: string) => void;
}

export interface UsePagerArgs
  extends Pick<UseListReturn<PagerParams, any, any>, "retrieve"> {
  pageSize?: number;
}

export const usePager = ({ retrieve, pageSize = 10 }: UsePagerArgs): UsePagerReturn => {
  const onFetchMore = useCallback(() => {
    retrieve(
      ({ offset = -pageSize, term }) => ({
        offset: offset + pageSize,
        limit: pageSize,
        term,
      }),
      { mode: "append" },
    );
  }, [retrieve, pageSize]);
  const onSearch = useCallback(
    (term: string) => retrieve({ term, offset: 0, limit: pageSize }),
    [retrieve, pageSize],
  );
  return { onFetchMore, onSearch };
};
