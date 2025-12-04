// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input, List } from "@synnaxlabs/pluto";
import { plural } from "pluralize";
import { type ReactElement, useCallback } from "react";

import { useContext } from "@/view/context";
import { type Query, type UseQueryReturn } from "@/view/useQuery";

export interface SearchProps<Q extends Query> extends UseQueryReturn<Q> {}

export const Search = <Q extends Query>({
  query,
  onQueryChange,
}: SearchProps<Q>): ReactElement => {
  const { resourceType } = useContext("View.Search");

  const handleSearch = useCallback(
    (searchTerm: string) => {
      onQueryChange((q) => ({ ...q, ...List.search(q, searchTerm) }));
    },
    [onQueryChange],
  );

  return (
    <Input.Text
      size="small"
      level="h5"
      variant="text"
      value={query.searchTerm ?? ""}
      onChange={handleSearch}
      placeholder={`Search ${plural(resourceType)}...`}
    />
  );
};
