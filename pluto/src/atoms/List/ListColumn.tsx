import { CSSProperties, useEffect, useState } from "react";

import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretUp } from "react-icons/ai";

import { measureTextWidth } from "../../util/canvas";
import { sortFunc } from "../../util/sort";

import { useListContext } from "./ListContext";
import {
  RenderableRecord,
  ListItemProps,
  TypedListColumn,
  TypedListTransform,
} from "./types";

import { Space } from "@/atoms/Space";
import { Text } from "@/atoms/Typography";
import { useFont } from "@/theming";

import "./ListColumn.css";

type SortState<E extends RenderableRecord<E>> = [keyof E | null, boolean];

export interface ListColumnHeaderProps<E extends RenderableRecord<E>> {
  columns: Array<TypedListColumn<E>>;
}

const ListColumnHeader = <E extends RenderableRecord<E>>({
  columns: initialColumns,
}: ListColumnHeaderProps<E>): JSX.Element => {
  const {
    columnar: { columns, setColumns },
    sourceData,
    setTransform,
    removeTransform,
  } = useListContext<E>();

  const font = useFont("p");
  const [sort, setSort] = useState<SortState<E>>([null, false]);

  const onSort = (k: keyof E): void => {
    const [prevSort, prevDir] = sort;
    if (prevSort === k) {
      if (!prevDir) {
        setSort([null, false]);
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
        columns.length === 0 ? initialColumns : columns,
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
          let endIcon;
          if (col.key === key) {
            endIcon = dir ? <AiFillCaretUp /> : <AiFillCaretDown />;
          }
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
        selected && "pluto-list-col-item__container--selected"
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
          <ListColumnValue key={col.key as string} entry={entry} col={col} />
        ))}
    </Space>
  );
};

interface ListColumnValueProps<E extends RenderableRecord<E>> {
  entry: E;
  col: TypedListColumn<E>;
}

const ListColumnValue = <E extends RenderableRecord<E>>({
  entry,
  col: { render: Render, width, ...col },
}: ListColumnValueProps<E>): JSX.Element => {
  const style: CSSProperties = { width, userSelect: "none", padding: 6 };
  if (Render != null) return <Render entry={entry} style={style} />;
  return (
    <Text
      key={col.key as string}
      level="p"
      style={{ minWidth: width, userSelect: "none", padding: 6 }}
    >
      {entry[col.key]}
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
  columns: Array<TypedListColumn<E>>,
  data: E[],
  font: string,
  padding = 60
): Array<TypedListColumn<E>> => {
  const le = longestEntries(data);
  return columns.map((col) => {
    const labelWidth = measureTextWidth(col.label, font);
    const entryWidth = measureTextWidth(le[col.key], font);
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
): TypedListTransform<E> => {
  return (data) => {
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
};
