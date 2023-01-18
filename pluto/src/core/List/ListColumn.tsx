// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSSProperties, useEffect, useState } from "react";

import {
  objectValueCompareFactory,
  convertRenderV,
  RenderableRecord,
} from "@synnaxlabs/x";
import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretUp } from "react-icons/ai";

import { Space } from "@/core/Space";
import { Text } from "@/core/Typography";
import { textWidth } from "@/core/Typography/textWidth";
import { useFont } from "@/theming";
import { ArrayTransform } from "@/util/transform";

import { useListContext } from "./ListContext";

import "./ListColumn.css";

import { ListItemProps, ListColumn as ListColumnT } from "./types";

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
    <Space direction="x" size="medium" className="pluto-list-col-header__container">
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
              {col.name}
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
      direction="x"
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
      {convertRenderV(entry[col.key])}
    </Text>
  );
};

const columnWidths = <E extends RenderableRecord<E>>(
  columns: Array<ListColumnT<E>>,
  data: E[],
  font: string,
  padding = 60
): Array<ListColumnT<E>> => {
  const le = longestEntries(data);
  return columns.map((col) => {
    const labelWidth = textWidth(col.name, font);
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

const sortTransform =
  <E extends RenderableRecord<E>>(k: keyof E, dir: boolean): ArrayTransform<E> =>
  (data: E[]) => {
    if (data.length === 0) return data;
    return [...data].sort(objectValueCompareFactory(k, data[0], !dir));
  };

export const ListColumn = {
  Header: ListColumnHeader,
  Item: ListColumnItem,
  itemHeight: 30,
};
