// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List as BaseList } from "./List";
import { ListColumn } from "./ListColumn";
import { useListContext } from "./ListContext";
import { ListCore } from "./ListCore";
import { ListSearch } from "./ListSearch";
export type { RenderableRecord, TypedListColumn, ListItemProps } from "./types";

type BaseListType = typeof BaseList;

interface ListType extends BaseListType {
  Column: typeof ListColumn;
  Search: typeof ListSearch;
  Core: typeof ListCore;
  useContext: typeof useListContext;
}

export const List = BaseList as ListType;

List.Column = ListColumn;
List.Search = ListSearch;
List.Core = ListCore;
List.useContext = useListContext;
