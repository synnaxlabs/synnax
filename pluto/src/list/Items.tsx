// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Items.css";

import { type record } from "@synnaxlabs/x";
import { memo, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useData } from "@/list/Frame";
import { type ItemRenderProp } from "@/list/Item";

export interface ItemsProps<K extends record.Key = record.Key> extends Omit<
  Flex.BoxProps,
  "children" | "ref"
> {
  children: ItemRenderProp<K>;
  emptyContent?: ReactNode;
  displayItems?: number;
}

const BaseItems = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  className,
  children,
  emptyContent,
  displayItems,
  style,
  direction,
  x,
  y,
  ...rest
}: ItemsProps<K>): ReactElement => {
  const { ref, getItems, getTotalSize, data, itemHeight, sentinelRef } = useData<
    K,
    E
  >();
  const visibleData = getItems();
  let content = emptyContent;
  const hasItems = data.length > 0;
  if (hasItems)
    content = (
      <div
        className={CSS.BE("list", "virtualizer")}
        style={{ minHeight: getTotalSize() }}
      >
        {visibleData.map(({ key, index, translate }) =>
          children({ key, index, itemKey: key, translate }),
        )}
        {sentinelRef != null && (
          <div
            ref={sentinelRef}
            className={CSS.BE("list", "sentinel")}
            aria-hidden="true"
          />
        )}
      </div>
    );

  let minHeight: number | undefined;
  if (itemHeight != null && displayItems != null && isFinite(displayItems) && hasItems)
    minHeight = Math.min(displayItems, visibleData.length) * itemHeight + 1;

  const parsedDirection = Flex.parseDirection(direction, x, y);
  return (
    <Flex.Box
      gap={0}
      ref={ref}
      className={CSS(
        className,
        CSS.BE("list", "items"),
        !hasItems && CSS.BEM("list", "items", "empty"),
      )}
      style={{ height: minHeight, ...style }}
      full={parsedDirection}
      direction={parsedDirection}
      {...rest}
    >
      {content}
    </Flex.Box>
  );
};

export const Items = memo(BaseItems) as typeof BaseItems;
