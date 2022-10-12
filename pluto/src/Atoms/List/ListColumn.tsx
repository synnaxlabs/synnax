import Space from "../Space/Space";
import Text from "../Typography/Text";
import TextWithIcon from "../Typography/TextWithIcon";
import { useFont } from "../../Theme/hooks";
import { useEffect, useState } from "react";
import { AiFillCaretDown, AiFillCaretUp } from "react-icons/ai";
import { sortFunc } from "../../util/sort";
import { useListContext } from "./ListContext";
import {
  Key,
  ListItemProps,
  TypedListColumn,
  TypedListEntry,
  TypedListTransform,
} from "./Types";
import clsx from "clsx";
import "./ListColumn.css";
import { getTextWidth } from "../../util/canvas";

type SortState<K extends Key, E extends TypedListEntry<K>> = [keyof E, boolean];

export interface ListColumnHeaderProps<
  K extends Key,
  E extends TypedListEntry<K>
> {
  columns: TypedListColumn<K, E>[];
}

const ListColumnHeader = <K extends Key, E extends TypedListEntry<K>>({
  columns: initialColumns,
}: ListColumnHeaderProps<K, E>) => {
  const {
    columnar: { columns, setColumns },
    sourceData,
    setTransform,
    removeTransform,
  } = useListContext<K, E>();

  const font = useFont("p");
  const [sort, setSort] = useState<SortState<K, E>>(["", false]);

  const onSort = (k: keyof E) => {
    const [prevSort, prevDir] = sort;
    if (prevSort == k) {
      if (!prevDir) {
        setSort(["", false]);
        removeTransform("sort");
      } else {
        setSort([k, !prevDir]);
        setTransform("sort", sortTransform(k, !prevDir));
      }
    } else {
      setSort([k, true]);
      setTransform("sort", sortTransform(k, true));
    }
  };

  useEffect(() => {
    setColumns((columns) => {
      return columnWidths(
        !columns.length ? initialColumns : columns,
        sourceData,
        font,
        60
      );
    });
  }, [sourceData, initialColumns]);

  return (
    <Space
      direction="horizontal"
      size="medium"
      className="pluto-list-col__header__container"
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map((col) => {
          const [key, dir] = sort;
          let endIcon = undefined;
          if (col.key === key) {
            endIcon = dir ? <AiFillCaretUp /> : <AiFillCaretDown />;
          }
          return (
            <TextWithIcon
              key={col.key as string}
              justify="spaceBetween"
              level="p"
              endIcon={endIcon}
              style={{
                width: col.width,
                cursor: "pointer",
                userSelect: "none",
                fontWeight: "bold",
              }}
              onClick={() => onSort(col.key)}
            >
              {col.label}
            </TextWithIcon>
          );
        })}
    </Space>
  );
};

export const ListColumnItem = <K extends Key, E extends TypedListEntry<K>>({
  entry,
  selected,
  columns,
  onSelect,
  ...props
}: ListItemProps<K, E>) => {
  return (
    <Space
      className={clsx(
        "pluto-list-col__item__container",
        selected && "pluto-list-col__item__container--selected"
      )}
      direction="horizontal"
      size="medium"
      onClick={() => onSelect(entry.key)}
      align="center"
      {...props}
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map((col) => (
          <Text
            key={col.key as string}
            level="p"
            style={{ width: col.width, userSelect: "none", padding: 6 }}
          >
            {entry[col.key]}
          </Text>
        ))}
    </Space>
  );
};

export const entrySortFunc =
  <K extends Key, E extends TypedListEntry<K>>(type: string, key: keyof E) =>
  (a: E, b: E) =>
    sortFunc(type)(a[key], b[key]);

const reverseSort = (f: (a: any, b: any) => number) => (a: any, b: any) =>
  f(b, a);

const columnWidths = <K extends Key, E extends TypedListEntry<K>>(
  columns: TypedListColumn<K, E>[],
  data: E[],
  font: string,
  padding: number = 60
): TypedListColumn<K, E>[] => {
  const le = longestEntries(data);
  return columns.map((col) => {
    const labelWidth = getTextWidth(col.label, font);
    const entryWidth = getTextWidth(le[col.key], font);
    return {
      ...col,
      width: Math.max(labelWidth, entryWidth) + padding,
    };
  });
};

const longestEntries = <K extends Key, E extends TypedListEntry<K>>(
  data: E[]
): Record<keyof E, string> => {
  const longest = {} as Record<keyof E, string>;
  data.forEach((entry: E) => {
    Object.entries(entry).map(([key, value]: [keyof E, string]) => {
      if (value.length > (longest[key]?.length || 0)) {
        longest[key] = value;
      }
    });
  });
  return longest;
};

const sortTransform = <K extends Key, E extends TypedListEntry<K>>(
  k: keyof E,
  dir: boolean
): TypedListTransform<K, E> => {
  return (data) => {
    if (data.length == 0) return data;
    const v = data[0][k];
    let sortF = entrySortFunc(typeof v, k);
    if (!dir) sortF = reverseSort(sortF);
    return [...data].sort(sortF);
  };
};

const Column = {
  Header: ListColumnHeader,
  Item: ListColumnItem,
};

export default Column;
