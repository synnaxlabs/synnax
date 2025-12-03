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

export const Search = (): ReactElement => {
  const { request, onRequestChange, resourceType } = useContext("Search");

  const handleSearch = useCallback(
    (searchTerm: string) => {
      onRequestChange((p) => ({ ...p, ...List.search(p, searchTerm) }));
    },
    [onRequestChange],
  );

  return (
    <Input.Text
      size="small"
      level="h5"
      variant="text"
      value={request.searchTerm ?? ""}
      onChange={handleSearch}
      placeholder={`Search ${plural(resourceType)}...`}
    />
  );
};
