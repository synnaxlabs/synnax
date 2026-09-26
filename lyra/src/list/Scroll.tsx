// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Scroll.css";

import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useData } from "@/list/Frame";
import { ScrollContext } from "@/list/scrollContext";

/** Props for {@link Scroll}. */
export interface ScrollProps extends Omit<Flex.BoxProps, "ref"> {
  /** Sizes the container to hold this many items before it scrolls. */
  displayItems?: number;
  /**
   * Smooths the height change when the item count changes. Set it only when the list
   * is sized by its content; a list sized by its container lags behind every resize.
   */
  animateHeight?: boolean;
}

/* The container's 1rem top and bottom padding (Scroll.css); the sized box is
   border-box, so omitting it leaves short lists scrolling by exactly this amount. */
const VERTICAL_PADDING = 12;

/**
 * The scroll container of a {@link Frame}. It holds {@link Items} and any content
 * rendered around them, and it is the element a virtualized list measures.
 */
export const Scroll = ({
  className,
  children,
  displayItems,
  animateHeight = false,
  style,
  direction,
  x,
  y,
  ...rest
}: ScrollProps): ReactElement => {
  const { ref, data, itemHeight, getTotalSize } = useData();
  const hasItems = data.length > 0;
  const isVirtual = getTotalSize() != null;

  let minHeight: number | undefined;
  if (itemHeight != null && displayItems != null && isFinite(displayItems) && hasItems)
    minHeight = Math.min(displayItems, data.length) * itemHeight + VERTICAL_PADDING + 1;

  const boxStyle = useMemo(
    () => ({
      height: minHeight,
      [CSS.variable("list-item-height")]:
        itemHeight != null ? `${itemHeight}px` : undefined,
      ...style,
    }),
    [minHeight, itemHeight, style],
  );

  const parsedDirection = Flex.parseDirection(direction, x, y);
  return (
    <Flex.Box
      gap={0}
      ref={ref}
      className={CSS.cls(
        className,
        CSS.BE("list", "scroll"),
        isVirtual && CSS.BEM("list", "scroll", "virtual"),
        !hasItems && CSS.BEM("list", "scroll", "empty"),
        animateHeight && CSS.BEM("list", "scroll", "animate-height"),
      )}
      style={boxStyle}
      full={parsedDirection}
      direction={parsedDirection}
      {...rest}
    >
      <ScrollContext value>{children}</ScrollContext>
    </Flex.Box>
  );
};
