import Space from "../Space/Space";
import { ColumnHeader, ColumnItem } from "./ColumnList";
import VirtualCore from "./CoreList";
import List from "./List";
import { Key, TypedColumn, TypedListEntry } from "./ListContext";
import SearchList from "./SearchList";

export interface ColumnListProps<K extends Key, E extends TypedListEntry<K>> {
  data: E[];
  columns: TypedColumn<K, E>[];
}

export const SelectableColumnSearchList = <
  K extends Key,
  E extends TypedListEntry<K>
>({
  data,
  columns,
}: ColumnListProps<K, E>) => {
  return (
    <List data={data}>
      <Space direction="vertical" empty>
        <SearchList />
        <Space
          empty
          style={{
            backgroundColor: "var(--pluto-gray-m3)",
            border: "1px solid var(--pluto-gray-m2)",
            borderTop: "none",
          }}
        >
          <ColumnHeader columns={columns} />
          <VirtualCore itemHeight={24}>
            {(props) => <ColumnItem {...props} />}
          </VirtualCore>
        </Space>
      </Space>
    </List>
  );
};
