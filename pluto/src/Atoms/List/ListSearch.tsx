import { useEffect } from "react";
import useSearch from "../../util/useSearch";
import Input from "../Input/Input";
import { useListContext } from "./ListContext";
import { Key, TypedListEntry } from "./Types";
import "./ListSearch.css";

export interface ListSearchProps<K extends Key, E extends TypedListEntry<K>> {
  Input?: React.ComponentType<{
    value?: string;
    onChange?: (value: string) => void;
    className?: string;
  }>;
}

export default function ListSearch<K extends Key, E extends TypedListEntry<K>>({
  Input: InputC = Input,
}: ListSearchProps<K, E>) {
  const [query, setQuery, search] = useSearch<E>();
  const { setTransform } = useListContext<K, E>();
  useEffect(() => setTransform("search", search), [search]);
  return (
    <InputC
      value={query}
      onChange={setQuery}
      className="pluto-list__search__input"
    />
  );
}
