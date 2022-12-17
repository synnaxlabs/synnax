import { useCallback, useState } from "react";

import Fuse from "fuse.js";

export const useSearch = <E extends Record<string, unknown>>(): [
  string,
  (value: string) => void,
  (data: E[]) => E[]
] => {
  const [query, setQuery] = useState("");
  const searchFunc = useCallback(
    (data: E[]) => {
      if (data?.length === 0) return data;
      const fuse = new Fuse(data, { keys: Object.keys(data[0]) });
      return query.length > 0 ? fuse.search(query).map((res) => res.item) : data;
    },
    [query]
  );
  return [query, setQuery, searchFunc];
};
