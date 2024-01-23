// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactElement, useRef } from "react";

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useContext } from "@/list/Context";
import { type ItemProps } from "@/list/types";
import { type RenderProp } from "@/util/renderProp";

import "@/list/Core.css";

export interface VirtualCoreProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  itemHeight: number;
  children: RenderProp<ItemProps<K, E>>;
  overscan?: number;
}

const VirtualCore = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  itemHeight,
  children,
  overscan = 5,
  className,
  ...props
}: VirtualCoreProps<K, E>): ReactElement => {
  if (itemHeight <= 0) throw new Error("itemHeight must be greater than 0");
  const {
    data,
    emptyContent,
    columnar: { columns },
    hover,
    select,
    infinite: { hasMore, onFetchMore },
  } = useContext<K, E>();
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemHeight,
    overscan,
  });

  const items = virtualizer.getVirtualItems();

  if (items.at(-1)?.index === data.length - 1 && hasMore) {
    onFetchMore?.();
  }

  return (
    <div
      ref={parentRef}
      className={CSS(className, CSS.BE("list", "container"))}
      {...props}
    >
      {data.length === 0 ? (
        emptyContent
      ) : (
        <div style={{ height: virtualizer.getTotalSize() }}>
          {items.map(({ index, start }) => {
            const entry = data[index];
            return children({
              key: entry.key,
              index,
              onSelect: select.onSelect,
              entry,
              columns,
              selected: select.value.includes(entry.key),
              hovered: index === hover.value,
              style: {
                transform: `translateY(${start}px)`,
                position: "absolute",
              },
            });
          })}
        </div>
      )}
    </div>
  );
};

export const Core = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>(
  props: Omit<Align.SpaceProps, "children"> & {
    children: RenderProp<ItemProps<K, E>>;
  },
): ReactElement => {
  const { data, emptyContent, columnar, hover, select } = useContext<K, E>();

  return (
    <Align.Space className={CSS.BE("list", "container")} {...props} empty>
      {data.length === 0 ? (
        emptyContent
      ) : (
        <div>
          {data.map((entry, index) =>
            props.children({
              key: entry.key.toString(),
              index,
              onSelect: select.onSelect,
              entry,
              columns: columnar.columns,
              selected: select.value.includes(entry.key),
              hovered: index === hover.value,
              style: {},
            }),
          )}
        </div>
      )}
    </Align.Space>
  );
};

export type CoreBaseType = typeof Core;

/**
 * Categorizes the core components for a list.
 *
 * @property Virtual - A virtualized renderer for a list.
 */
export interface CoreType extends CoreBaseType {
  /**
   * A virtualized list core.
   *
   * @param props - Props for the virtualized list core. All props not defined below
   * are passed to the underlying div element wrapping the list.
   * @param props.itemHeight - The height of each item in the list.
   * @param props.overscan - The number of extra items to render during virtualization.
   * This improves scroll performance, but also adds extra DOM elements, so make sure
   * to set this to a reasonable value.
   * @param props.children - A render props that renders each item in the list. It should
   * implement the {@link ItemProps} interface. The virtualizer will pass all props
   * satisfying this interface to the render props.
   */
  Virtual: typeof VirtualCore;
}

Core.Virtual = VirtualCore;
