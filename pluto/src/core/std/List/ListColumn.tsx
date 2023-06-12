// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSSProperties, ReactElement, useEffect, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  Compare,
  convertRenderV,
  Key,
  KeyedRenderableRecord,
  RenderableRecord,
} from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { useListContext } from "@/core/std/List/ListContext";
import { ListItemProps, ListColumn as ListColumnT } from "@/core/std/List/types";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/core/std/Menu/ContextMenu";
import { Space } from "@/core/std/Space";
import { Text } from "@/core/std/Typography";
import { Theming } from "@/core/theming";
import { ArrayTransform } from "@/util/transform";

import "@/core/std/List/ListColumn.css";

type SortState<E extends RenderableRecord> = [keyof E | null, boolean];

export interface ListColumnHeaderProps<
  K extends Key,
  E extends KeyedRenderableRecord<K>
> {
  columns: Array<ListColumnT<K, E>>;
}

const SORT_TRANSFORM = "sort";

const ListColumnHeader = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  columns: initialColumns,
}: ListColumnHeaderProps<K, E>): ReactElement => {
  const {
    columnar: { columns, setColumns },
    sourceData,
    setTransform,
    deleteTransform,
  } = useListContext<K, E>();

  const font = Theming.useFont("p");
  const [sort, setSort] = useState<SortState<E>>([null, false]);

  const onSort = (k: keyof E): void => {
    const [prevSort, prevDir] = sort;
    if (prevSort === k) {
      if (!prevDir) {
        setSort([null, false]);
        deleteTransform(SORT_TRANSFORM);
      } else {
        setSort([k, !prevDir]);
        setTransform(SORT_TRANSFORM, sortTransform(k, !prevDir));
      }
    } else {
      setSort([k, true]);
      setTransform(SORT_TRANSFORM, sortTransform(k, true));
    }
  };

  useEffect(() => {
    setColumns((prev) =>
      columnWidths(prev.length === 0 ? initialColumns : prev, sourceData, font)
    );
  }, [font, sourceData, initialColumns]);

  return (
    <Space
      direction="x"
      size="medium"
      className={CSS.BE("list-col-header", "container")}
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map(({ key, cWidth: width, name }) => {
          const [sortKey, dir] = sort;
          let endIcon;
          const entry = sourceData[0];
          if (key === sortKey) endIcon = dir ? <Icon.Caret.Up /> : <Icon.Caret.Down />;
          return (
            <Text.WithIcon
              className={CSS.BE("list-col-header", "item")}
              key={key.toString()}
              justify="spaceBetween"
              level="p"
              endIcon={endIcon}
              style={{ width }}
              shrink={false}
              onClick={() => entry != null && key in entry && onSort(key as keyof E)}
            >
              {name}
            </Text.WithIcon>
          );
        })}
    </Space>
  );
};

const ListColumnItem = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  entry,
  selected,
  columns,
  onSelect,
  index,
  ...props
}: ListItemProps<K, E>): ReactElement => {
  const handleSelect = (): void => onSelect?.(entry.key);
  return (
    <Space
      id={entry.key.toString()}
      className={CSS(
        CONTEXT_TARGET,
        CSS.BE("list-col-item", "container"),
        onSelect != null && CSS.BEM("list-col-item", "container", "selectable"),
        selected && CSS.BEM("list-col-item", "container", "selected"),
        selected && CONTEXT_SELECTED
      )}
      direction="x"
      onClick={handleSelect}
      onContextMenu={handleSelect}
      align="center"
      {...props}
      size="medium"
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map((col) => (
          <ListColumnValue key={col.key.toString()} entry={entry} col={col} />
        ))}
    </Space>
  );
};

interface ListColumnValueProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {
  entry: E;
  col: ListColumnT<K, E>;
}

const ListColumnValue = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  entry,
  col: { width, ...col },
}: ListColumnValueProps<K, E>): ReactElement | null => {
  const style: CSSProperties = { width: col.cWidth, userSelect: "none", padding: 6 };
  if (col.render != null) return col.render({ key: col.key, entry, style });
  let rv: E[keyof E] | string;
  if (col.stringer != null) rv = col.stringer(entry);
  else rv = entry[col.key as keyof E];
  return (
    <Text key={col.key.toString()} level="p" style={style}>
      {convertRenderV(rv)}
    </Text>
  );
};

const columnWidths = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>(
  columns: Array<ListColumnT<K, E>>,
  data: E[],
  font: string
): Array<ListColumnT<K, E>> => {
  const le = longestEntries(data, columns);
  return columns.map((col) => {
    if (col.width != null) col.cWidth = col.width;
    else {
      const { width: labelWidth } = Text.dimensions(col.name, font);
      const { width: entryWidth } = Text.dimensions(le[col.key as keyof E], font);
      col.cWidth = Math.max(labelWidth, entryWidth);
    }
    return col;
  });
};

const longestEntries = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>(
  data: E[],
  columns: Array<ListColumnT<K, E>>
): Record<keyof E, string> => {
  const longest = {} as const as Record<keyof E, string>;
  data.forEach((entry: E) => {
    columns.forEach(({ key, stringer }) => {
      const rv = entry[key as keyof E];
      if (rv == null) return;
      const value = stringer != null ? stringer(entry) : rv;
      if (typeof value === "string" && value.length > longest[key as keyof E]?.length) {
        longest[key as keyof E] = value;
      }
    });
  });
  return longest;
};

const sortTransform =
  <
    K extends Key = Key,
    E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
  >(
    k: keyof E,
    dir: boolean
  ): ArrayTransform<E> =>
  (data: E[]) => {
    if (data.length === 0) return data;
    return [...data].sort(Compare.newFieldF(k, data[0], !dir));
  };

export const ListColumn = {
  /**
   * The header for a column list.
   *
   * @param columns - The columns to render. See {@link ListColumnT}.
   */
  Header: ListColumnHeader,
  /**
   * The item to use for a column list. This should be used as the child render prop
   * in a list render implmentation e.g. {@link List.Core.Virtual}.
   *
   * @param props - implements the {@link ListItemProps} interface. All these props
   * should be provided by the list render implementation.
   */
  Item: ListColumnItem,
  /** The default height of a column list item. */
  itemHeight: 30,
};
