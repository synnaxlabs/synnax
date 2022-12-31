// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentType, useEffect } from "react";

import { useListContext } from "./ListContext";
import { RenderableRecord } from "./types";

import { Input as DefaultInput, InputProps } from "@/atoms/Input";
import { useSearch } from "@/hooks";
import "./ListSearch.css";

export interface ListSearchProps {
  Input?: ComponentType<InputProps>;
}

export const ListSearch = <E extends RenderableRecord<E>>({
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
