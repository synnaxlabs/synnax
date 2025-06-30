// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Core.css";

import { bounds, type record } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  type ComponentPropsWithoutRef,
  type FC,
  memo,
  type ReactElement,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
} from "react";

import { CSS } from "@/css";
import { usePrevious } from "@/hooks/ref";
import { useData } from "@/list/Data";
import { useSelectionContext, useSelectionUtils } from "@/list/Selector";
import { type ItemProps, type ItemRenderProp } from "@/list/types";

export interface ItemsProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  itemHeight: number;
  children: ItemRenderProp<K, E>;
  overscan?: number;
}

export const Items = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  itemHeight,
  children,
  overscan = 0,
  className,
  ...rest
}: ItemsProps<K, E>): ReactElement => {
  const data = useData<K, E>();
  const { selected, hover } = useSelectionContext();
  const { onSelect, setHover } = useSelectionUtils();
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemHeight,
    overscan,
  });

  const prev = usePrevious(data.items);

  // Whenever the data changes, scroll to the top of the list
  useLayoutEffect(() => {
    if (prev == null || prev.length === 0) return;
    if (data.items.length > 0 && data.items[0] !== prev[0]) {
      virtualizer?.scrollToIndex(0);
      setHover(0);
    }
  }, [data.items]);

  useEffect(() => {
    if (parentRef.current == null) return;
    const rng = virtualizer.calculateRange();
    if (rng == null) return;
    const b = bounds.construct(rng.startIndex + 2, rng.endIndex - 2);
    if (bounds.contains(b, hover)) return;
    virtualizer.scrollToIndex(hover);
  }, [hover, itemHeight]);

  const vItems = virtualizer.getVirtualItems();
  const lastItemIndex = vItems.at(-1)?.index;
  const dataLength = data.items.length;
  // useEffect(() => {
  //   if (lastItemIndex === dataLength - 1) onFetchMore?.();
  // }, [lastItemIndex, dataLength]);

  if (itemHeight <= 0) throw new Error("itemHeight must be greater than 0");

  const empty = data.items.length === 0;

  const ChildComponent = useMemo((): FC<
    Omit<ItemProps<K, E>, "entry"> & { itemKey: K }
  > => {
    const C = memo(
      ({
        itemKey,
        index,
        onSelect,
        selected,
        hovered,
        translate,
      }: Omit<ItemProps<K, E>, "entry"> & { itemKey: K }) => {
        const entry = data.useItem(itemKey);
        if (entry == null) return null;
        return children({
          key: itemKey,
          entry,
          index,
          onSelect,
          selected,
          hovered,
          translate,
        });
      },
    );
    C.displayName = "Item";
    return C;
  }, [data.items, children]);

  return (
    <div
      ref={parentRef}
      className={CSS(
        className,
        CSS.BE("list", "container"),
        empty && CSS.BM("list", "empty"),
      )}
      {...rest}
    >
      <div
        className={CSS.BE("list", "virtualizer")}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {vItems.map(({ index, start }) => {
          const key = data.items[index];
          return (
            <ChildComponent
              key={key}
              itemKey={key}
              index={index}
              onSelect={onSelect}
              selected={selected.includes(key)}
              hovered={index === hoverValue}
              translate={start}
            />
          );
        })}
      </div>
    </div>
  );
};
