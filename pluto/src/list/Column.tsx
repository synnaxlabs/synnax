// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Column.css";

import { Icon } from "@synnaxlabs/media";
import {
  type ArrayTransform,
  compare,
  convertRenderV,
  type Key,
  type Keyed,
  type RenderableValue,
} from "@synnaxlabs/x";
import {
  createContext,
  type CSSProperties,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useState,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useRequiredContext } from "@/hooks/useRequiredContext";
import { useDataUtilContext, useSourceData } from "@/list/Data";
import { ItemFrame, type ItemFrameProps } from "@/list/Item";
import { type ItemProps } from "@/list/types";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { type RenderProp } from "@/util/renderProp";

type RenderF<K extends Key = Key, E extends Keyed<K> = Keyed<K>> = RenderProp<{
  key: string | number | symbol;
  entry: E;
  style: CSSProperties;
}>;

export interface ColumnSpec<K extends Key = Key, E extends Keyed<K> = Keyed<K>> {
  /** The key of the object to render. */
  key: keyof E | string;
  /** A custom render function for each item in the colummn. */
  render?: RenderF<K, E>;
  stringer?: (entry: E) => string;
  /** The name/title of the column. */
  name: string;
  /** Whether the column is visible by default. */
  visible?: boolean;
  /**
   * The width of the column in pixels. Used to structure the list as a table.
   * If not provided, the column will be sized to fit the content. This should
   * always be specified when the render function is provided.
   */
  width?: number;
  cWidth?: number;
  shade?: Text.Shade;
  weight?: Text.Weight;
}

interface ColumnContextValue<K extends Key = Key, E extends Keyed<K> = Keyed<K>> {
  columns: Array<ColumnSpec<K, E>>;
}

export const ColumnContext = createContext<ColumnContextValue | null>(null);

const useColumnContext = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): ColumnContextValue<K, E> => useRequiredContext(ColumnContext);

type SortState<E> = [keyof E | null, boolean];

export interface ColumnHeaderProps<K extends Key, E extends Keyed<K>>
  extends PropsWithChildren<{}> {
  hide?: boolean;
  columns: Array<ColumnSpec<K, E>>;
}

const SORT_TRANSFORM = "sort";

const Header = <K extends Key, E extends Keyed<K>>({
  hide = false,
  columns: initialColumns,
  children,
}: ColumnHeaderProps<K, E>): ReactElement => {
  const sourceData = useSourceData<K, E>();
  const { setTransform, deleteTransform } = useDataUtilContext<K, E>();

  const font = Theming.useTypography("p").toString();
  const [sort, setSort] = useState<SortState<E>>([null, false]);
  const [ctxValue, setCtxValue] = useState<ColumnContextValue<K, E>>({
    columns: initialColumns,
  });

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
    setCtxValue((prev) => ({
      columns: columnWidths(
        prev.columns.length === 0 ? initialColumns : prev.columns,
        sourceData,
        font,
      ),
    }));
  }, [font, sourceData, initialColumns]);

  return (
    <ColumnContext.Provider value={ctxValue as ColumnContextValue}>
      <Align.Space
        direction="x"
        size="medium"
        className={CSS(CSS.BE("list-col-header", "container"), CSS.visible(!hide))}
      >
        {ctxValue.columns
          .filter(({ visible = true }) => visible)
          .map(({ key, cWidth: width, name }) => {
            const [sortKey, dir] = sort;
            let endIcon;
            const entry = sourceData[0];
            if (key === sortKey)
              endIcon = dir ? <Icon.Caret.Up /> : <Icon.Caret.Down />;
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
      </Align.Space>
      {children}
    </ColumnContext.Provider>
  );
};

const Item = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  entry,
  onSelect,
  className,
  ...props
}: ItemProps<K, E> & ItemFrameProps<K, E>): ReactElement => {
  const { columns } = useColumnContext<K, E>();
  return (
    <ItemFrame<K, E>
      key={entry.key.toString()}
      entry={entry}
      onSelect={onSelect}
      className={CSS(
        className,
        CSS.BE("list-col-item", "container"),
        onSelect != null && CSS.BEM("list-col-item", "container", "selectable"),
      )}
      align="center"
      size="medium"
      {...props}
    >
      {columns
        .filter(({ visible = true }) => visible)
        .map((col) => (
          <ListColumnValue key={col.key.toString()} entry={entry} col={col} />
        ))}
    </ItemFrame>
  );
};

interface ListColumnValueProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>> {
  entry: E;
  col: ColumnSpec<K, E>;
}

const ListColumnValue = <K extends Key, E extends Keyed<K>>({
  entry,
  col: { width, ...col },
}: ListColumnValueProps<K, E>): ReactElement | null => {
  const style: CSSProperties = {
    width: col.cWidth,
    userSelect: "none",
    padding: "1rem",
    flexShrink: 0,
  };
  if (col.render != null) return col.render({ key: col.key, entry, style });
  let rv: E[keyof E] | string;
  if (col.stringer != null) rv = col.stringer(entry);
  else rv = entry[col.key as keyof E];
  return (
    <Text.Text
      className={CSS.BE("list-col-item-value", col.key.toString())}
      key={col.key.toString()}
      level="p"
      style={style}
      shade={col.shade}
      weight={col.weight}
    >
      {convertRenderV(rv as RenderableValue)}
    </Text.Text>
  );
};

const columnWidths = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>(
  columns: Array<ColumnSpec<K, E>>,
  data: E[],
  font: string,
  padding: number = 30,
): Array<ColumnSpec<K, E>> => {
  const le = longestEntries(data, columns);
  return columns.map((col) => {
    if (col.width != null) col.cWidth = col.width;
    else {
      const { width: labelWidth } = Text.dimensions(col.name, font);
      const { width: entryWidth } = Text.dimensions(le[col.key as keyof E], font);
      col.cWidth = Math.max(labelWidth, entryWidth) + padding;
    }
    return col;
  });
};

const longestEntries = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>(
  data: E[],
  columns: Array<ColumnSpec<K, E>>,
): Record<keyof E, string> => {
  const longest = {} as const as Record<keyof E, string>;
  data.forEach((entry: E) =>
    columns.forEach(({ key, stringer }) => {
      const rv = entry[key as keyof E];
      if (rv == null) return;
      const value = stringer != null ? stringer(entry) : rv.toString();
      if (!(key in longest) || value.length > longest[key as keyof E].length)
        longest[key as keyof E] = value;
    }),
  );
  return longest;
};

const sortTransform =
  <K extends Key = Key, E extends Keyed<K> = Keyed<K>>(
    k: keyof E,
    dir: boolean,
  ): ArrayTransform<E> =>
  ({ data }) => {
    if (data.length === 0) return { data, transformed: false };
    return {
      data: [...data].sort(compare.newFieldF(k, data[0], !dir)),
      transformed: true,
    };
  };

export const Column = {
  /**
   * The header for a column list.
   *
   * @param columns - The columns to render. See {@link ListColumnT}.
   */
  Header,
  /**
   * The item to use for a column list. This should be used as the child render prop
   * in a list render implementation e.g. {@link List.Core.Virtual}.
   *
   * @param props - implements the {@link ItemProps} interface. All these props
   * should be provided by the list render implementation.
   */
  Item,
  /** The default height of a column list item. */
  itemHeight: 36,
};
