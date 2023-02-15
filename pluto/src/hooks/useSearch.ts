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

import { proxyMemo } from "@/memo";
import { ArrayTransform } from "@/util/transform";

/** Props for the {@link useSearchTransform} hook. */
export interface UseSearchTransformProps<E extends UnknownRecord<E>> {
  query: string;
  opts?: Fuse.IFuseOptions<E>;
}

const defaultOpts: Fuse.IFuseOptions<UnknownRecord<UnknownRecord>> = {
  threshold: 0.3,
};

/**
 * @returns a transform that can be used to filter an array of objects in memory
 * based on a search query.
 *
 * Can be used in conjunction with `useTransform` to add search functionality
 * alongside other transforms.
 *
 * Uses fuse.js under the hood.
 *
 * @param query - The query to search for.
 * @param opts - The options to pass to the Fuse.js search. See the Fuse.js
 * documentation for more information on these options.
 */
export const useSearchTransform = <E extends UnknownRecord<E>>({
  query,
  opts,
}: UseSearchTransformProps<E>): ArrayTransform<E> =>
  useCallback(
    proxyMemo((data: E[]) => {
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
