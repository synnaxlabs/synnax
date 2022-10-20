import { ComponentType, useEffect } from "react";
import { useSearch } from "../../Hooks";
import { Input as DefaultInput, InputProps } from "../Input";
import { useListContext } from "./ListContext";
import { Key, TypedListEntry } from "./Types";
import "./ListSearch.css";

export interface ListSearchProps<K extends Key, E extends TypedListEntry<K>> {
  Input?: ComponentType<InputProps>;
}

export default function ListSearch<K extends Key, E extends TypedListEntry<K>>({
  Input = DefaultInput,
}: ListSearchProps<K, E>) {
  const [query, setQuery, search] = useSearch<E>();
  const { setTransform } = useListContext<K, E>();
  useEffect(() => setTransform("search", search), [search]);
  return (
    <Input
      value={query}
      onChange={(e) => setQuery(e.target.value)}
      className="pluto-list__search__input"
    />
  );
}
