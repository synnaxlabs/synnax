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
