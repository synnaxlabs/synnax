import { CSSProperties, useEffect, useState } from "react";

import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretUp } from "react-icons/ai";

import { useListContext } from "./ListContext";
import { ListItemProps, ListColumn as ListColumnT } from "./types";

import { Space } from "@/atoms/Space";
import { Text } from "@/atoms/Typography";
import { useFont } from "@/theming";
import { textWidth } from "@/util/canvas";
import { RenderableRecord } from "@/util/record";
import { render } from "@/util/renderable";
import { sortFunc } from "@/util/sort";
import { ArrayTransform } from "@/util/transform";

import "./ListColumn.css";

type SortState<E extends RenderableRecord<E>> = [keyof E | null, boolean];

export interface ListColumnHeaderProps<E extends RenderableRecord<E>> {
  columns: Array<ListColumnT<E>>;
}

const ListColumnHeader = <E extends RenderableRecord<E>>({
  columns: initialColumns,
}: ListColumnHeaderProps<E>): JSX.Element => {
  const {
    columnar: { columns, setColumns },
    sourceData,
    setTransform,
    deleteTransform,
  } = useListContext<E>();

  const font = useFont("p");
  const [sort, setSort] = useState<SortState<E>>([null, false]);

  const onSort = (k: keyof E): void => {
    const [prevSort, prevDir] = sort;
    if (prevSort === k) {
      if (!prevDir) {
        setSort([null, false]);
        deleteTransform("sort");
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
    setColumns((prev) =>
      columnWidths(prev.length === 0 ? initialColumns : prev, sourceData, font, 60)
    );
  }, [sourceData, initialColumns]);

  return (
    <Space
      direction="horizontal"
      size="medium"
      className="pluto-list-col-header__container"
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map((col) => {
          const [key, dir] = sort;
          let endIcon;
          if (col.key === key) endIcon = dir ? <AiFillCaretUp /> : <AiFillCaretDown />;
          return (
            <Text.WithIcon
              key={col.key as string}
              justify="spaceBetween"
              level="p"
              endIcon={endIcon}
              style={{
                minWidth: col.width,
                cursor: "pointer",
                userSelect: "none",
                fontWeight: "bold",
              }}
              onClick={() => onSort(col.key)}
            >
              {col.label}
            </Text.WithIcon>
          );
        })}
    </Space>
  );
};

const ListColumnItem = <E extends RenderableRecord<E>>({
  entry,
  selected,
  columns,
  onSelect,
  ...props
}: ListItemProps<E>): JSX.Element => {
  return (
    <Space
      className={clsx(
        "pluto-list-col-item__container",
        onSelect != null && "pluto-list-col-item__container--selectable",
        selected && "pluto-list-col-item__container--selected"
      )}
      direction="horizontal"
      size="medium"
      onClick={() => onSelect?.(entry.key)}
      align="center"
      {...props}
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map((col) => (
          <ListColumnValue key={col.key as string} entry={entry} col={col} />
        ))}
    </Space>
  );
};

interface ListColumnValueProps<E extends RenderableRecord<E>> {
  entry: E;
  col: ListColumnT<E>;
}

const ListColumnValue = <E extends RenderableRecord<E>>({
  entry,
  col: { width, ...col },
}: ListColumnValueProps<E>): JSX.Element | null => {
  const style: CSSProperties = { width, userSelect: "none", padding: 6 };
  if (col.render != null) return col.render({ key: col.key, entry, style });
  return (
    <Text
      key={col.key as string}
      level="p"
      style={{ minWidth: width, userSelect: "none", padding: 6 }}
    >
      {render(entry[col.key])}
    </Text>
  );
};

const entrySortFunc =
  <E extends RenderableRecord<E>>(type: string, key: keyof E) =>
  (a: E, b: E) =>
    sortFunc(type)(a[key], b[key]);

const reverseSort =
  <T,>(f: (a: T, b: T) => number) =>
  (a: T, b: T) =>
    f(b, a);

const columnWidths = <E extends RenderableRecord<E>>(
  columns: Array<ListColumnT<E>>,
  data: E[],
  font: string,
  padding = 60
): Array<ListColumnT<E>> => {
  const le = longestEntries(data);
  return columns.map((col) => {
    const labelWidth = textWidth(col.label, font);
    const entryWidth = textWidth(le[col.key], font);
    return {
      ...col,
      width: Math.max(labelWidth, entryWidth) + padding,
    };
  });
};

const longestEntries = <E extends RenderableRecord<E>>(
  data: E[]
): Record<keyof E, string> => {
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  const longest = {} as Record<keyof E, string>;
  data.forEach((entry: E) => {
    Object.entries(entry).forEach(([key, value]) => {
      if (
        typeof value === "string" &&
        value.length > (longest[key as keyof E]?.length !== 0 || 0)
      ) {
        longest[key as keyof E] = value;
      }
    });
  });
  return longest;
};

const sortTransform = <E extends RenderableRecord<E>>(
  k: keyof E,
  dir: boolean
): ArrayTransform<E> => {
  return (data: E[]) => {
    if (data.length === 0) return data;
    const v = data[0][k];
    let sortF = entrySortFunc(typeof v, k);
    if (!dir) sortF = reverseSort(sortF);
    return [...data].sort(sortF);
  };
};

export const ListColumn = {
  Header: ListColumnHeader,
  Item: ListColumnItem,
  itemHeight: 30,
};
