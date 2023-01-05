// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List as CoreList } from "./List";
import { ListColumn } from "./ListColumn";
import { useListContext } from "./ListContext";
import { ListCore } from "./ListCore";
import { ListSearch } from "./ListSearch";
import { ListSelector } from "./ListSelector";
export type { ListSelectorProps } from "./ListSelector";
export type { ListColumnHeaderProps } from "./ListColumn";

export type { ListColumn, ListItemProps } from "./types";

type CoreListType = typeof CoreList;

interface ListType extends CoreListType {
  Column: typeof ListColumn;
  Search: typeof ListSearch;
  Core: typeof ListCore;
  Selector: typeof ListSelector;
  useContext: typeof useListContext;
}

export const List = CoreList as ListType;

List.Column = ListColumn;
List.Search = ListSearch;
List.Core = ListCore;
List.useContext = useListContext;
List.Selector = ListSelector;
