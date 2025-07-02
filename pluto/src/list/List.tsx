// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Core.css";

import { type record } from "@synnaxlabs/x";
import { useVirtualizer, type Virtualizer } from "@tanstack/react-virtual";
import {
  type ComponentPropsWithoutRef,
  type ReactElement,
  type RefObject,
  useRef,
} from "react";

import { CSS } from "@/css";
import { type ItemRenderProp } from "@/list/Item";

export interface UseProps<K extends record.Key = record.Key> {
  data: K[];
  itemHeight?: number;
}

export interface UseReturn {
  ref: RefObject<HTMLDivElement | null>;
  virtualizer: Virtualizer<HTMLDivElement, Element>;
}

export interface ListProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends Omit<ComponentPropsWithoutRef<"div">, "children">,
    UseReturn {
  data: K[];
  children: ItemRenderProp<K, E>;
  useItem: (key: K) => E;
  onFetchMore?: () => void;
}

export const use = <K extends record.Key = record.Key>({
  data,
  itemHeight = 30,
}: UseProps<K>): UseReturn => {
  const ref = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => ref.current,
    estimateSize: () => itemHeight,
  });
  return { ref, virtualizer };
};

export const List = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  virtualizer,
  data,
  className,
  ref,
  children,
  useItem,
  ...rest
}: ListProps<K, E>): ReactElement => {
  const visibleData = virtualizer.getVirtualItems();
  return (
    <div ref={ref} className={CSS(className, CSS.BE("list", "container"))} {...rest}>
      <div
        className={CSS.BE("list", "virtualizer")}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {visibleData.map(({ index, start }) => {
          const key = data[index];
          return (
            children({ key, index, translate: start, useItem, itemKey: key }) ?? null
          );
        })}
      </div>
    </div>
  );
};
