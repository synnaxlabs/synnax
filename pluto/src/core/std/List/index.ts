// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List as CoreList } from "@/core/std/List/List";
import { ListColumn } from "@/core/std/List/ListColumn";
import { useListContext } from "@/core/std/List/ListContext";
import { ListCore } from "@/core/std/List/ListCore";
import { ListSearch } from "@/core/std/List/ListSearch";
import { ListSelector } from "@/core/std/List/ListSelector";

export type { ListSelectorProps } from "@/core/std/List/ListSelector";
export type { ListColumnHeaderProps } from "@/core/std/List/ListColumn";
export type { ListProps } from "@/core/std/List/List";
export type { ListColumn, ListItemProps } from "@/core/std/List/types";

type CoreListType = typeof CoreList;

interface ListType extends CoreListType {
  /**
   * Categorizes the column related components for a list.
   *
   * @property Header - The header for a column in a list.
   * @property Item - The individual item rnederer for a column list.
   * @property itemHeight - The height for each item in a column list.
   */
  Column: typeof ListColumn;
  /**
   * Implements in-client search for a list.
   *
   * @param props - The props for the List.Search component.
   * @param props.children - A custom input render prop for the search functionality. This
   * must implement the InputControl<string> interface.
   * @param opts - Custom options for the search functionality. See the {@link fuse.IFuseOptions}
   * interface for more details.
   */
  Search: typeof ListSearch;
  /**
   * Categorizes the core components for a list.
   *
   * @property Virtual - A virtualized renderer for a list.
   */
  Core: typeof ListCore;
  /**
   * Implements selection behavior for a list.
   *
   * @param props - The props for the List.Selector component. These props are identical
   * to the props for {@link useSelectMultiple} hook.
   */
  Selector: typeof ListSelector;
  /**
   * A hook to access the context information for a list. This hook should only be used
   * when you know what you are doing, and are looking to extend the functionality of a
   * list component.
   */
  useContext: typeof useListContext;
}

/**
 * The main component for building a List. By itself, it does not render any HTML, and
 * should be used in conjunction with its subcomponents (List.'X') to build a list
 * component to fit your needs.
 *
 * @param props - The props for the List component.
 * @param props.data - The data to be displayed in the list. The values of the object in
 * each entry of the array must satisfy the {@link RenderableValue} interface i.e. they
 * must be a primitive type or implement a 'toString' method.
 * @param props.children - Sub-components of the List component to add additional functionality.
 *
 */
export const List = CoreList as ListType;

List.Column = ListColumn;
List.Search = ListSearch;
List.Core = ListCore;
List.useContext = useListContext;
List.Selector = ListSelector;
