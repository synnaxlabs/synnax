// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnknownRecord, ArrayTransform } from "@synnaxlabs/x";
import Fuse from "fuse.js";

import { proxyMemo } from "@/core/memo";

/** Props for the {@link createSearchTransform} hook. */
export interface UseSearchTransformProps<E extends UnknownRecord<E>> {
  term: string;
  searcher?: Searcher<E> | ((data: E[]) => Searcher<E>);
}

export interface Searcher<E extends UnknownRecord<E>> {
  search: (term: string) => E[];
}

const defaultOpts: Fuse.IFuseOptions<UnknownRecord<UnknownRecord>> = {
  threshold: 0.3,
};

export const fuseSearcher =
  (opts?: Fuse.IFuseOptions<UnknownRecord>) =>
  <E extends UnknownRecord<E>>(data: E[]): Searcher<E> => {
    const fuse = new Fuse(data, {
      keys: Object.keys(data[0]),
      ...defaultOpts,
      ...opts,
    });
    return {
      search: (term: string) => fuse.search(term).map(({ item }) => item),
    };
  };

const defaultSearcher = fuseSearcher();

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
export const createSearchTransform = <E extends UnknownRecord<E>>({
  term,
  searcher = defaultSearcher<E>,
}: UseSearchTransformProps<E>): ArrayTransform<E> =>
  proxyMemo((data) => {
    if (typeof searcher === "function") {
      if (term.length === 0 || data?.length === 0) return data;
      return searcher(data).search(term);
    }
    return searcher.search(term);
  });
