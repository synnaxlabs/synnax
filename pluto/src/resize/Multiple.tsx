// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, direction, math, type xy } from "@synnaxlabs/x";
import {
  Children,
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Core } from "@/resize/Core";

/** Props for the {@link Resize.Multiple} component. */
export interface MultipleProps extends Flex.BoxProps {
  sizeDistribution: number[];
  onDragHandle: (e: ResizeStartEvent, i: number) => void;
}

export type ResizeStartEvent = xy.Client & { preventDefault: () => void };

/** Props for the {@link Resize.useMultiple} hook. */
export interface UseMultipleProps {
  count: number;
  onResize?: (sizes: number[]) => void;
  direction?: direction.Crude;
  initialSizes?: number[];
  minSize?: number;
}

/** Return value of the {@link Resize.useMultiple} hook. */
export interface UseMultipleReturn {
  setSize: (i: number, size?: number) => void;
  props: Pick<MultipleProps, "sizeDistribution" | "onDragHandle" | "direction"> & {
    ref: RefObject<HTMLDivElement | null>;
  };
}

interface MultipleState {
  sizeDistribution: number[];
  root: number | null;
}

/**
 * A hook that implements the control logic for {@link Resize.Multiple}. This hook
 * should be used in conjunction with {@link Resize.Multiple}.
 *
 * @param props - The a component props.
 * @param props.count - The number of panels to render. This should be the same as the
 * number of children passed to {@link Resize.Multiple}.
 * @param props.onResize - A callback executed when the panels are resized.
 * @param props.initialSizes - The initial sizes of the panels. This should be an array of
 * numbers, where each number is the initial size of the corresponding panel either in pixels
 * or as a decimal percentage. If this array is not provided (or is shorter than the expected length),
 *  the (remaining) panels will be evenly distributed across the container.
 * @param props.direction - The direction in which the panels should be draggable. This should
 * be Default: "x"
 * @param props.minSize - The smallest size the panels can be resized to. Defaults to 100.
 *
 * @returns The props that should be passed to {@link Resize.Multiple} along with a few
 * utility functions. setSize can be used to manually set the size of a particular panel.
 */
export const useMultiple = ({
  onResize,
  count,
  initialSizes = [],
  minSize = 100,
  direction = "x",
}: UseMultipleProps): UseMultipleReturn => {
  const ref = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<MultipleState>({
    sizeDistribution: calculateInitialSizeDistribution(initialSizes, count),
    root: null,
  });

  const _handleResize = useCallback(
    (dragging: number, clientPos?: number, targetSize?: number) => {
      if (ref.current == null) return;
      const parentSize = box.dim(box.construct(ref.current), direction);
      setState((prev) =>
        handleResize(prev, parentSize, dragging, minSize, clientPos, targetSize),
      );
    },
    [minSize, setState, direction],
  );

  useEffect(
    () => onResize?.(state.sizeDistribution),
    [state.sizeDistribution, onResize],
  );

  const handleDragHandle = useCallback(
    (e: ResizeStartEvent, dragging: number): void => {
      e.preventDefault();
      const dim = direction === "x" ? "clientX" : "clientY";
      const handleMouseMove = (e: MouseEvent): void => _handleResize(dragging, e[dim]);
      const handleMouseUp = (): void => {
        setState((prev) => ({ ...prev, root: null }));
        document.removeEventListener("mousemove", handleMouseMove);
      };
      setState((prev) => ({ ...prev, root: e[dim] }));
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp, { once: true });
    },
    [setState, direction, handleResize],
  );

  const setSize = useCallback(
    (i: number, size?: number) => _handleResize(i, undefined, size),
    [handleResize],
  );

  return {
    setSize,
    props: {
      direction,
      sizeDistribution: state.sizeDistribution,
      onDragHandle: handleDragHandle,
      ref,
    },
  };
};

/**
 * A set of panels that can be resized within their container. Resize.Multiple must be
 * used in conjunction with {@link Resize.useMultiple}.
 *
 * @param props - The component props. All unused props will be passed to the div
 * containing the div containing the panels. Generally these props should not be provided
 * directly, and you should instead spread the props returned from {@link Resize.useMultiple}.
 * The only exceptions to this are stylistic props (e.g. className, style, etc.) as well
 * as the `children` prop.
 */
export const Multiple = ({
  ref,
  direction: direction_ = "x",
  children: _children,
  sizeDistribution,
  onDragHandle: onDrag,
  className,
  ...rest
}: MultipleProps) => {
  const dir = direction.construct(direction_);
  const children = Children.toArray(_children);

  /**
   * You may be wondering, why on earth are we doing this? Well, the answer is that
   * if you're moving elements within the resize multiple, and the sizes are all the
   * same, any resize observers will not trigger. The slight offset ensures that if
   * a re-ordering occurs, the resize observers will trigger and everything will
   * be in sync.
   */
  sizeDistribution = slightlyOffsetEvenDistribution(sizeDistribution);

  return (
    <Flex.Box
      {...rest}
      ref={ref}
      direction={dir}
      className={CSS(CSS.B("resize-multiple"), className)}
      empty
      grow
    >
      {children.map((child, i) => (
        <Core
          onDragStart={(e) => onDrag(e, i)}
          key={i}
          location={direction.location(dir)}
          size={sizeDistribution[i] * 100}
          sizeUnits="%"
          showHandle={i !== children.length - 1}
        >
          {child}
        </Core>
      ))}
    </Flex.Box>
  );
};

const calculateInitialSizeDistribution = (
  initial: number[],
  count: number,
): number[] => {
  const total = initial.reduce((a, b) => a + b, 0);
  const gap = count - initial.length;
  if (gap <= 0) return initial.slice(0, count).map((v) => v / total);
  if (initial.every((v) => v <= 1)) {
    const remaining = 1 - total;
    return [...initial, ...Array(gap).fill(remaining / gap)];
  }
  return initial.map(() => 1 / count);
};

const DELTA = 0.001;

const slightlyOffsetEvenDistribution = (sizes: number[]): number[] => {
  if (sizes.every((v) => math.closeTo(v, 1 / sizes.length), DELTA))
    return sizes.map((v, i) => {
      if (i % 2 === 0) return v + DELTA;
      if (i === sizes.length - 1) return v;
      return v - DELTA;
    });
  return sizes;
};

const handleResize = (
  prev: MultipleState,
  parentSize: number,
  dragging: number,
  minSize: number,
  clientPos?: number,
  targetSize?: number,
): MultipleState => {
  const diffPercentage = calculateDiffPercentage(
    prev,
    parentSize,
    dragging,
    clientPos,
    targetSize,
  );
  const [sizeDistribution, changed] = resizeWithSibling(
    prev.sizeDistribution,
    dragging,
    diffPercentage,
    minSize / parentSize,
  );
  const root = changed ? (clientPos ?? null) : prev.root;
  return { ...prev, sizeDistribution, root };
};

export const calculateDiffPercentage = (
  prev: MultipleState,
  parentSize: number,
  dragging: number,
  clientPos?: number,
  targetSize?: number,
): number => {
  let diff: number;
  // If the caller provided a target size, prefer that.
  if (targetSize != null) {
    // If the target size is a pixel value, convert it to a percentage.
    if (targetSize > 1) targetSize /= parentSize;
    diff = targetSize - prev.sizeDistribution[dragging];
  } else if (clientPos != null) {
    if (prev.root === null) throw new Error("resize root is null during handle drag");
    diff = (clientPos - prev.root) / parentSize;
  } else throw new Error("must provide either a MouseEvent or a targetSize");
  return diff;
};

const resizeWithSibling = (
  prevDistribution: number[],
  item: number,
  diff: number,
  minSizePercent: number,
): [number[], boolean] => {
  let next = item;

  // The while loops find a pane we can actually resize.
  while (next > 0 && prevDistribution[next] + diff <= minSizePercent) next--;
  const nextSize = prevDistribution[next] + diff;

  const nextSibling = findResizableSibling(
    next,
    prevDistribution,
    minSizePercent,
    diff,
  );
  const nextSiblingSize = prevDistribution[nextSibling] - diff;
  // This means we can't resize any panes so we return null to indicate that the next
  // set of sizes should remain the same.
  if (nextSize < minSizePercent || nextSiblingSize < minSizePercent)
    return [prevDistribution, false];

  return [
    prevDistribution.map((p, i) => {
      if (i === next) return nextSize;
      if (i === nextSibling) return nextSiblingSize;
      return p;
    }),
    true,
  ];
};

const findResizableSibling = (
  item: number,
  sizes: number[],
  minSize: number,
  diff: number,
): number => {
  const f = item === sizes.length - 1 ? findBackward : findForward;
  return f(item, sizes, minSize, diff);
};

const findForward = (
  start: number,
  sizes: number[],
  minSize: number,
  diff: number,
): number => {
  let i = start + 1;
  while (i < sizes.length - 1 && sizes[i] - diff <= minSize) i++;
  return i;
};

const findBackward = (
  start: number,
  sizes: number[],
  minSize: number,
  diff: number,
): number => {
  let i = start - 1;
  while (i > 0 && sizes[i] - diff <= minSize) i--;
  return i;
};
