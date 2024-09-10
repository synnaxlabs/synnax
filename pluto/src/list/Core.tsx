// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Core.css";

import { bounds, type Key, type Keyed } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  type ComponentPropsWithoutRef,
  type ReactElement,
  useEffect,
  useLayoutEffect,
  useRef,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { usePrevious } from "@/hooks/ref";
import { useDataContext } from "@/list/Data";
import { useHoverContext } from "@/list/Hover";
import { useInfiniteContext } from "@/list/Infinite";
import { useSelection, useSelectionContext, useSelectionUtils } from "@/list/Selector";
import { type ItemRenderProp } from "@/list/types";

export interface VirtualCoreProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  itemHeight: number;
  children: ItemRenderProp<K, E>;
  overscan?: number;
}

const scrollToRelevantChild = (
  hover: number,
  prevHover: number,
  ref: HTMLDivElement,
) => {
  const dirMultiplier = hover > prevHover ? 1 : -1;
  let scrollTo = hover;
  const idealHover = hover + dirMultiplier * 2;
  if (bounds.contains({ lower: 0, upper: ref.children.length }, idealHover))
    scrollTo = hover + dirMultiplier * 2;
  else scrollTo = hover;
  ref.children[scrollTo]?.scrollIntoView({
    block: "nearest",
    inline: "nearest",
    behavior: "smooth",
  });
};

const VirtualCore = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  itemHeight,
  children,
  overscan = 5,
  className,
  ...props
}: VirtualCoreProps<K, E>): ReactElement => {
  const { hasMore, onFetchMore } = useInfiniteContext();
  const { hover: hoverValue, setHover } = useHoverContext();
  const {
    transformedData: data,
    emptyContent,
    transformed,
    sourceData,
  } = useDataContext<K, E>();
  const selected = useSelection();
  const { onSelect } = useSelectionUtils();
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemHeight,
    overscan,
  });

  const prev = usePrevious(data);

  // Whenever the data changes, scroll to the top of the list
  useLayoutEffect(() => {
    if (prev == null || prev.length === 0) return;
    if (data.length > 0 && data[0].key !== prev[0].key) {
      virtualizer?.scrollToIndex(0);
      setHover(0);
    }
  }, [data]);

  useEffect(() => {
    if (parentRef.current == null) return;
    const rng = virtualizer.calculateRange();
    if (rng == null) return;
    const b = bounds.construct(rng.startIndex + 2, rng.endIndex - 2);
    if (bounds.contains(b, hoverValue)) return;
    virtualizer.scrollToIndex(hoverValue);
  }, [hoverValue, itemHeight]);

  const items = virtualizer.getVirtualItems();
  const lastItemIndex = items.at(-1)?.index;
  const dataLength = data.length;
  useEffect(() => {
    if (lastItemIndex === dataLength - 1 && hasMore) onFetchMore?.();
  }, [lastItemIndex, dataLength, hasMore]);

  if (itemHeight <= 0) throw new Error("itemHeight must be greater than 0");

  const empty = data.length === 0;

  return (
    <div
      ref={parentRef}
      className={CSS(
        className,
        CSS.BE("list", "container"),
        empty && CSS.BM("list", "empty"),
      )}
      {...props}
    >
      {empty ? (
        emptyContent
      ) : (
        <div
          className={CSS.BE("list", "virtualizer")}
          style={{ height: virtualizer.getTotalSize() }}
        >
          {items.map(({ index, start }) => {
            const entry = data[index];
            let sourceIndex = index;
            if (transformed)
              sourceIndex = sourceData.findIndex((e) => e.key === entry.key);
            return children({
              key: entry.key,
              sourceIndex,
              index,
              onSelect,
              entry,
              selected: selected.includes(entry.key),
              hovered: index === hoverValue,
              translate: start,
            });
          })}
        </div>
      )}
    </div>
  );
};

export const Core = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  itemHeight = 50,
  className,
  ...props
}: Omit<Align.SpaceProps, "children"> & {
  children: ItemRenderProp<K, E>;
  itemHeight?: number;
}): ReactElement => {
  const {
    transformedData: data,
    transformed,
    emptyContent,
    sourceData,
  } = useDataContext<K, E>();
  const { hover } = useHoverContext();
  const { selected } = useSelectionContext();
  const { onSelect } = useSelectionUtils();

  // let's assume the itemHeight is a 'rough item height', but is still useful for
  // detecting when the user hovers over a particular element. If the current `hover`
  // value multiplied by the itemHeight is greater than the total height of the list,
  // then we can assume the user is hovering over or past the last element in the list,
  // and we should scroll to the bottom of the list.
  const ref = useRef<HTMLDivElement>(null);
  const prevHover = usePrevious(hover) ?? 0;
  useLayoutEffect(() => {
    if (ref.current == null) return;
    scrollToRelevantChild(hover, prevHover, ref.current);
  }, [hover, itemHeight]);

  return (
    <Align.Space
      className={CSS(className, CSS.BE("list", "container"))}
      ref={ref}
      size={0}
      {...props}
    >
      {data.length === 0 ? (
        emptyContent
      ) : (
        <>
          {data.map((entry, index) => {
            let sourceIndex = index;
            if (transformed)
              sourceIndex = sourceData.findIndex((e) => e.key === entry.key);
            return props.children({
              key: entry.key,
              index,
              sourceIndex,
              onSelect,
              entry,
              selected: selected.includes(entry.key),
              hovered: index === hover,
            });
          })}
        </>
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
