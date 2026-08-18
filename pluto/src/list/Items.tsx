// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Items.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useMemo } from "react";

import { memo } from "@/component/memo";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useData } from "@/list/Frame";
import { type ItemRenderProp } from "@/list/Item";

/** Props for {@link Items}. */
export interface ItemsProps<K extends record.Key = record.Key> extends Omit<
  Flex.BoxProps,
  "children" | "ref"
> {
  /** Renders one item. It is called once per visible key. */
  children: ItemRenderProp<K>;
  /** Rendered in place of the items when the list is empty. */
  emptyContent?: ReactNode;
  /** Sizes the list to hold this many items before it scrolls. */
  displayItems?: number;
  /**
   * Smooths the height change when the item count changes. Set it only when the list
   * is sized by its content; a list sized by its container lags behind every resize.
   */
  animateHeight?: boolean;
}

/* The container's 1rem top and bottom padding (Items.css); the sized box is
   border-box, so omitting it leaves short lists scrolling by exactly this amount. */
const VERTICAL_PADDING = 12;

const BaseItems = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  className,
  children,
  emptyContent,
  displayItems,
  animateHeight = false,
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
  const totalSize = getTotalSize();
  const isVirtual = totalSize != null;
  const virtualizerStyle = useMemo(() => ({ minHeight: totalSize }), [totalSize]);
  if (hasItems)
    content = (
      <div className={CSS.BE("list", "virtualizer")} style={virtualizerStyle}>
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
    minHeight = Math.min(displayItems, data.length) * itemHeight + VERTICAL_PADDING + 1;

  const boxStyle = useMemo(
    () => ({
      height: minHeight,
      [CSS.var("list-item-height")]: itemHeight != null ? `${itemHeight}px` : undefined,
      ...style,
    }),
    [minHeight, itemHeight, style],
  );

  const parsedDirection = Flex.parseDirection(direction, x, y);
  return (
    <Flex.Box
      gap={0}
      ref={ref}
      className={CSS.cx(
        className,
        CSS.BE("list", "items"),
        isVirtual && CSS.BEM("list", "items", "virtual"),
        !hasItems && CSS.BEM("list", "items", "empty"),
        animateHeight && CSS.BEM("list", "items", "animate-height"),
      )}
      style={boxStyle}
      full={parsedDirection}
      direction={parsedDirection}
      {...rest}
    >
      {content}
    </Flex.Box>
  );
};

/**
 * The scroll container for a {@link Frame}. It renders the visible items, handles
 * virtualization, and shows `emptyContent` when there are none.
 */
export const Items = memo(BaseItems);
