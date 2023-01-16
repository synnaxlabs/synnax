// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { UnknownRecord } from "@synnaxlabs/x";
import Fuse from "fuse.js";
import memoize from "proxy-memoize";

import { ArrayTransform } from "@/util/transform";

export interface UseSearchProps<E extends UnknownRecord<E>> {
  query: string;
  opts?: Fuse.IFuseOptions<E>;
}

const defaultOpts: Fuse.IFuseOptions<UnknownRecord<UnknownRecord>> = {
  threshold: 0.3,
};

export const useSearch = <E extends UnknownRecord<E>>({
  query,
  opts,
}: UseSearchProps<E>): ArrayTransform<E> => {
  return useCallback(
    memoize((data: E[]) => {
      if (data?.length === 0 || query.length === 0) return data;
      const fuse = new Fuse(data, {
        keys: Object.keys(data[0]),
        ...opts,
        ...defaultOpts,
      });
      return fuse.search(query).map((res) => res.item);
    }),
    [query]
  );
};
