// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, direction, math } from "@synnaxlabs/x";
import {
  Children,
  type ReactElement,
  type ReactNode,
  useCallback,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { Flex } from "@/flex";
import { Base } from "@/resize/Base";

/** Props for the {@link Split} component. */
export interface SplitProps extends Omit<
  Flex.BoxProps,
  "size" | "onResize" | "children"
> {
  /** The two panes to lay out, the first on the left/top and the second on the
   * right/bottom. Exactly two children are required. */
  children: [ReactNode, ReactNode];
  /**
   * The fraction of the container occupied by the first pane, as a decimal between 0 and
   * 1. Acts as the source of truth whenever the handle is not being actively dragged;
   * while a drag is in progress the panes render their transient drag ratio instead.
   * Defaults to 0.5.
   */
  size?: number;
  /** The smallest size, in pixels, either pane can be resized to. Defaults to 100. */
  minSize?: number;
  /**
   * Called continuously while the handle is being dragged with the first pane's live
   * fraction (0..1). Use for transient, per-frame side effects; do not persist from here.
   */
  onResize?: (size: number) => void;
  /**
   * Called once when a drag gesture ends with the committed fraction (0..1). This is the
   * callback to persist the new ratio to the source of truth.
   */
  onResizeEnd?: (size: number) => void;
}

/**
 * Offset applied to the rendered pane sizes when the split is evenly distributed. Two
 * panes at exactly 50% do not change size when their contents are swapped (e.g. a tab
 * reorder), so ResizeObservers inside them never fire. Nudging the panes by DELTA forces
 * a measurable size change on swap. Applied at render only; never reported via onResize.
 */
const DELTA = 0.001;

const HALF_SPLIT = 0.5;

/**
 * A pair of panes that can be resized relative to one another by dragging the handle
 * between them. Both panes are sized as percentages of the container, so the split ratio
 * is preserved as the container resizes.
 *
 * @param props - The component props. Unlisted props are forwarded to the underlying
 * {@link Flex.Box}. Exactly two children should be provided; the first is placed on the
 * left/top and the second on the right/bottom.
 */
export const Split = ({
  onResize,
  onResizeEnd,
  children,
  className,
  size = HALF_SPLIT,
  minSize = 100,
  align = "stretch",
  direction: propsDirection,
  x,
  y,
  pack,
  ...rest
}: SplitProps): ReactElement => {
  const dir = Flex.parseDirection(propsDirection, x, y, pack) ?? "x";
  const loc = direction.location(dir);
  const ref = useRef<HTMLDivElement>(null);
  const [dragSize, setDragSize] = useState<number | null>(null);
  const rendered = dragSize ?? size;
  const start = useRef(size);
  const parentSize = useRef(0);
  const [first, last] = Children.toArray(children);

  const handleStart = useCallback(() => {
    start.current = size;
    if (ref.current != null)
      parentSize.current = box.dim(box.construct(ref.current), dir);
  }, [size, dir]);

  const calcNextSize = useCallback(
    (region: box.Box): number | null => {
      const total = parentSize.current;
      if (total === 0) return null;
      const min = Math.min(minSize / total, 0.5);
      const diff = box.dim(region, dir, true) / total;
      return bounds.clamp({ lower: min, upper: 1 - min }, start.current + diff);
    },
    [dir, minSize],
  );

  const handleMove = useCallback(
    (region: box.Box) => {
      const next = calcNextSize(region);
      if (next == null) return;
      setDragSize(next);
      onResize?.(next);
    },
    [calcNextSize, onResize],
  );

  const handleEnd = useCallback(
    (region: box.Box) => {
      const next = calcNextSize(region);
      setDragSize(null);
      if (next != null) onResizeEnd?.(next);
    },
    [calcNextSize, onResizeEnd],
  );

  const handleDragStart = Cursor.useDrag({
    onStart: handleStart,
    onMove: handleMove,
    onEnd: handleEnd,
  });

  const offset = math.closeTo(rendered, HALF_SPLIT, DELTA) ? DELTA : 0;
  return (
    <Flex.Box
      {...rest}
      ref={ref}
      direction={dir}
      align={align}
      className={CSS(CSS.B("resize-split"), className)}
      empty
      grow
    >
      <Base
        location={loc}
        size={rendered + offset}
        decimal
        onPointerDown={handleDragStart}
      >
        {first}
      </Base>
      <Base location={loc} size={1 - rendered - offset} decimal hideHandle>
        {last}
      </Base>
    </Flex.Box>
  );
};
