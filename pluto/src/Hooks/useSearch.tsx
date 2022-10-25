import Fuse from "fuse.js";
import { useCallback, useState } from "react";

export const useSearch = <E extends Record<string, any>>(): [
  string,
  (value: string) => void,
  (data: E[]) => E[]
] => {
  const [query, setQuery] = useState("");
  const searchFunc = useCallback(
    (data: E[]) => {
      if (!data || data.length == 0) return data;
      const fuse = new Fuse(data, { keys: Object.keys(data[0]) });
      return query ? fuse.search(query).map((res) => res.item) : data;
    },
    [query]
  );
  return [query, setQuery, searchFunc];
};
