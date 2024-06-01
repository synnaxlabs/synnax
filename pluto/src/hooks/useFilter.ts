// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ArrayTransform,
  type Key,
  type Keyed,
  type TermSearcher,
} from "@synnaxlabs/x";
import Fuse, { type IFuseOptions } from "fuse.js";

import { proxyMemo } from "@/memo";

/** Props for the {@link createFilterTransform} function. */
export interface CreateFilterTransformProps<K extends Key, E extends Keyed<K>> {
  term: string;
  searcher?: TermSearcher<string, K, E> | ((data: E[]) => TermSearcher<string, K, E>);
}

const defaultOpts: IFuseOptions<unknown> = {
  threshold: 0.3,
};

export const fuseFilter =
  (opts?: IFuseOptions<unknown>) =>
  <K extends Key, E extends Keyed<K>>(data: E[]): TermSearcher<string, K, E> => {
    const fuse = new Fuse(data, {
      keys: Object.keys(data[0]),
      ...defaultOpts,
      ...opts,
    });
    return {
      type: "fuse",
      page: (page: number, perPage: number) =>
        data.slice(page * perPage, (page + 1) * perPage),
      search: (term: string) => fuse.search(term).map(({ item }) => item),
      retrieve: (keys: K[]) => data.filter((item) => keys.includes(item.key)),
    };
  };

const defaultFilter = fuseFilter();

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
export const createFilterTransform = <K extends Key, E extends Keyed<K>>({
  term,
  searcher = defaultFilter<K, E>,
}: CreateFilterTransformProps<K, E>): ArrayTransform<E> =>
  proxyMemo(({ data }) => {
    if (typeof searcher === "function") {
      if (term.length === 0 || data?.length === 0) return { data, transformed: false };
      return { data: searcher(data).search(term), transformed: true };
    }
    return { data: searcher.search(term), transformed: true };
  });
