import { List as CoreList } from "./List";
import { ListColumn } from "./ListColumn";
import { useListContext } from "./ListContext";
import { ListCore } from "./ListCore";
import { ListSearch } from "./ListSearch";
import { ListSelector } from "./ListSelector";
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
