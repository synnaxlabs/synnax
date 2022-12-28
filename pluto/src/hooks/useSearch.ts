import { useCallback } from "react";

import Fuse from "fuse.js";
import memoize from "proxy-memoize";

import { UnknownRecord } from "@/util/record";
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
