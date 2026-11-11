// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { plural } from "pluralize";
import { type ReactElement } from "react";

import { useContext } from "@/platform/view/context";

/** The search field of a frame, which holds the term. */
export const SearchInput = (): ReactElement | null => {
  const { editable, resourceType, searchInViews, searchTerm, setSearchTerm } =
    useContext("View.Search");
  if (!editable) return null;
  return (
    <Input.Text
      size="small"
      level="h5"
      color={9}
      startContent={startContent}
      variant="text"
      grow={searchInViews}
      value={searchTerm}
      onChange={setSearchTerm}
      placeholder={`Search ${plural(resourceType)}...`}
    />
  );
};

const startContent = <Icon.Search color={9} />;

/**
 * Searches the resources of the view, in place. It renders nothing when the frame leads
 * the views bar with the search.
 */
export const Search = (): ReactElement | null => {
  const { searchInViews } = useContext("View.Search");
  return searchInViews ? null : <SearchInput />;
};
