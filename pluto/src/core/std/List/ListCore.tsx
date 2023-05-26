// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement, useRef } from "react";

import { KeyedRenderableRecord } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";

import { CSS } from "@/core/css";
import { SelectedRecord } from "@/core/hooks/useSelectMultiple";
import { useListContext } from "@/core/std/List/ListContext";

import "@/core/std/List/ListCore.css";

import { ListItemProps } from "@/core/std/List/types";
import { RenderProp } from "@/util/renderProp";

export interface ListVirtualCoreProps<E extends KeyedRenderableRecord<E>>
  extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  itemHeight: number;
  children: RenderProp<ListItemProps<E>>;
  overscan?: number;
}

const ListVirtualCore = <E extends KeyedRenderableRecord<E>>({
  itemHeight,
  children,
  overscan = 5,
  ...props
}: ListVirtualCoreProps<E>): ReactElement => {
  if (itemHeight <= 0) throw new Error("itemHeight must be greater than 0");
  const {
    data,
    emptyContent,
    columnar: { columns },
    select: { onSelect },
  } = useListContext<E>();
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemHeight,
    overscan,
  });

  return (
    <div ref={parentRef} className={CSS.BE("list", "container")} {...props}>
      {data.length === 0 ? (
        emptyContent
      ) : (
        <div style={{ height: virtualizer.getTotalSize() }}>
          {virtualizer.getVirtualItems().map(({ index, start }) => {
            const entry = data[index];
            return children({
              key: index,
              index,
              onSelect,
              entry,
              columns,
              selected: (entry as SelectedRecord<E>)?.selected ?? false,
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

export const ListCore = {
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
   * implement the {@link ListItemProps} interface. The virtualizer will pass all props
   * satisfying this interface to the render props.
   */
  Virtual: ListVirtualCore,
};
