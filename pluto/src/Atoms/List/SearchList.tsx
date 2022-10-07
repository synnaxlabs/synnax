import Fuse from "fuse.js";
import { HTMLAttributes, useCallback } from "react";
import { useEffect } from "react";
import { useState } from "react";
import Input from "../Input/Input";
import Space from "../Space/Space";
import { Key, TypedListEntry, useListContext } from "./ListContext";

export interface SearchListProps {
  Input?: React.ComponentType<{
    value: string;
    onChange: (value: string) => void;
  }>;
}

export default function SearchList<K extends Key, E extends TypedListEntry<K>>({
  Input: AInput = Input,
}: SearchListProps) {
  const { setTransform } = useListContext<K, E>();
  const [search, setSearch] = useState("");

  const transform = useCallback(
    (data: E[]) => {
      const fuse = new Fuse(data, {
        keys: Object.keys(data[0]),
      });
      return search
        ? fuse
            .search(search)
            .map((res) => res.item)
            .slice(0, 10)
        : data;
    },
    [search]
  );

  useEffect(() => {
    setTransform("search", transform);
  }, [transform]);

  return <AInput value={search} onChange={setSearch} />;
}
