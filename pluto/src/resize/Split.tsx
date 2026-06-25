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
  useEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useCursorDrag } from "@/hooks/useCursorDrag";
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
   * The fraction of the container occupied by the first pane on mount, as a decimal
   * between 0 and 1. Defaults to 0.5.
   */
  initialSize?: number;
  /** The smallest size, in pixels, either pane can be resized to. Defaults to 100. */
  minSize?: number;
  /** Called with the first pane's fraction (0..1) whenever the panes are resized. */
  onResize?: (size: number) => void;
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
  children,
  className,
  initialSize = HALF_SPLIT,
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
  const [size, setSize] = useState(initialSize);
  const start = useRef(initialSize);
  const [first, last] = Children.toArray(children);

  useEffect(() => onResize?.(size), [size, onResize]);

  const handleStart = useCallback(
    () =>
      setSize((prev) => {
        start.current = prev;
        return prev;
      }),
    [],
  );

  const handleMove = useCallback(
    (region: box.Box) => {
      if (ref.current == null) return;
      const parentSize = box.dim(box.construct(ref.current), dir);
      if (parentSize === 0) return;
      const min = Math.min(minSize / parentSize, 0.5);
      const diff = box.dim(region, dir, true) / parentSize;
      setSize(bounds.clamp({ lower: min, upper: 1 - min }, start.current + diff));
    },
    [dir, minSize],
  );

  const handleDragStart = useCursorDrag({ onStart: handleStart, onMove: handleMove });

  const offset = math.closeTo(size, HALF_SPLIT, DELTA) ? DELTA : 0;
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
      <Base location={loc} size={size + offset} decimal onDragStart={handleDragStart}>
        {first}
      </Base>
      <Base location={loc} size={1 - size - offset} decimal hideHandle>
        {last}
      </Base>
    </Flex.Box>
  );
};
