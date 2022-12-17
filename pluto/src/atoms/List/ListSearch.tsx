import { ComponentType, useEffect } from "react";

import { useListContext } from "./ListContext";
import { ListEntry } from "./types";

import { Input as DefaultInput, InputProps } from "@/atoms/Input";
import { useSearch } from "@/hooks";
import "./ListSearch.css";

export interface ListSearchProps {
  Input?: ComponentType<InputProps>;
}

export const ListSearch = <E extends ListEntry>({
  Input = DefaultInput,
}: ListSearchProps): JSX.Element => {
  const [query, setQuery, search] = useSearch<E>();
  const { setTransform } = useListContext<E>();
  useEffect(() => setTransform("search", search), [search]);
  return (
    <Input
      value={query}
      onChange={(e) => setQuery(e.target.value)}
      className="pluto-list-search__input"
    />
  );
};
