// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Children,
  ForwardedRef,
  forwardRef,
  useCallback,
  useState,
  DragEvent as RDragEvent,
  MouseEvent as RMouseEvent,
  RefObject,
  useEffect,
} from "react";

import { UnexpectedError } from "@synnaxlabs/client";
import clsx from "clsx";

import { ResizeCore } from "./ResizeCore";

import { Space, SpaceProps } from "@/core/Space";
import {
  Box,
  Direction,
  getDirectionalSize,
  locationFromDirection,
  useResize,
} from "@/spatial";

export interface UseResizeMultipleProps {
  count: number;
  onResize?: (sizes: number[]) => void;
  direction?: Direction;
  initialSizes?: number[];
  minSize?: number;
}

export interface ResizeMultipleProps extends SpaceProps {
  sizeDistribution: number[];
  parentSize: number;
  onDragHandle: (e: RDragEvent | RMouseEvent, i: number) => void;
}

export interface UseResizeMultipleReturn {
  setSize: (i: number, size?: number) => void;
  props: ResizeMultipleProps & {
    ref: RefObject<HTMLDivElement>;
  };
}

interface ResizeMultipleState {
  sizeDistribution: number[];
  parentSize: number | null;
  root: number | null;
}

export const useResizeMultiple = ({
  count,
  onResize,
  initialSizes = [],
  minSize = 100,
  direction = "horizontal",
}: UseResizeMultipleProps): UseResizeMultipleReturn => {
  const [state, setState] = useState<ResizeMultipleState>({
    sizeDistribution: calculateInitialSizeDistribution(initialSizes, count),
    parentSize: null,
    root: null,
  });

  const _handleResize = useCallback(
    (dragging: number, clientPos?: number, targetSize?: number) => {
      setState((prev) => handleResize(prev, dragging, minSize, clientPos, targetSize));
    },
    [minSize, setState]
  );

  useEffect(
    () => onResize?.(state.sizeDistribution),
    [state.sizeDistribution, onResize]
  );

  const handleDragHandle = useCallback(
    (e: RDragEvent | RMouseEvent, dragging: number): void => {
      const dim = direction === "horizontal" ? "clientX" : "clientY";
      const handleMouseMove = (e: MouseEvent): void => _handleResize(dragging, e[dim]);
      const handleMouseUp = (): void => {
        setState((prev) => ({ ...prev, root: null }));
        document.removeEventListener("mousemove", handleMouseMove);
      };
      setState((prev) => ({ ...prev, root: e[dim] }));
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp, { once: true });
    },
    [setState, direction, handleResize]
  );

  const setSize = useCallback(
    (i: number, size?: number) => _handleResize(i, undefined, size),
    [handleResize]
  );

  const _handleParentResize = useCallback(
    (box: Box) =>
      setState((prev) => handleParentResize(prev, box, direction, count, minSize)),
    [direction, count, minSize]
  );

  const ref = useResize<HTMLDivElement>(_handleParentResize, { triggers: [direction] });

  return {
    setSize,
    props: {
      direction,
      sizeDistribution: state.sizeDistribution,
      parentSize: state.parentSize ?? 0,
      onDragHandle: handleDragHandle,
      ref,
    },
  };
};

export const ResizeMultiple = forwardRef(
  (
    {
      direction = "horizontal",
      children: _children,
      sizeDistribution,
      onDragHandle: onDrag,
      className,
      parentSize,
      ...props
    }: ResizeMultipleProps,
    ref: ForwardedRef<HTMLDivElement>
  ) => {
    const children = Children.toArray(_children);
    const location = locationFromDirection(direction);

    return (
      <Space
        ref={ref}
        direction={direction}
        className={clsx("pluto-resize-multiple", className)}
        empty
        grow
        {...props}
      >
        {children.map((child, i) => (
          <ResizeCore
            onDragStart={(e) => onDrag(e, i)}
            key={i}
            location={location}
            size={sizeDistribution[i] * parentSize}
            showHandle={i !== children.length - 1}
          >
            {child}
          </ResizeCore>
        ))}
      </Space>
    );
  }
);
ResizeMultiple.displayName = "ResizeMultiple";

export const calculateInitialSizeDistribution = (
  initial: number[],
  count: number
): number[] => {
  const total = initial.reduce((a, b) => a + b, 0);
  const gap = count - initial.length;
  if (gap <= 0) {
    return initial.slice(0, count).map((v) => v / total);
  }
  if (initial.every((v) => v <= 1)) {
    const remaining = 1 - total;
    return [...initial, ...Array(gap).fill(remaining / gap)];
  }
  return initial.map(() => 1 / count);
};

export const handleResize = (
  prev: ResizeMultipleState,
  dragging: number,
  minSize: number,
  clientPos?: number,
  targetSize?: number
): ResizeMultipleState => {
  if (prev.parentSize === null) return prev;
  const diffPercentage = calculateDiffPercentage(prev, dragging, clientPos, targetSize);
  const [sizeDistribution, changed] = resizeWithSibling(
    prev.sizeDistribution,
    dragging,
    diffPercentage,
    minSize / prev.parentSize
  );
  const root = changed ? clientPos ?? null : prev.root;
  return { ...prev, sizeDistribution, root };
};

export const handleParentResize = (
  prev: ResizeMultipleState,
  box: Box,
  direction: Direction,
  count: number,
  minSize: number
): ResizeMultipleState => {
  const nextParentSize = getDirectionalSize(direction, box);
  if (prev.parentSize == null && prev.sizeDistribution.length !== count)
    return {
      ...prev,
      parentSize: nextParentSize,
      sizeDistribution: Array.from({ length: count }, (_, i) => 1 / count),
    };
  prev = { ...prev, parentSize: nextParentSize };
  const nextSizes = distribute(prev, count, minSize);
  return { ...prev, parentSize: nextParentSize, sizeDistribution: nextSizes };
};

export const calculateDiffPercentage = (
  prev: ResizeMultipleState,
  dragging: number,
  clientPos?: number,
  targetSize?: number
): number => {
  if (prev.parentSize === null)
    throw new UnexpectedError("parent size is null during handle drag");
  let diff: number;
  // If the caller provided a target size, prefer that.
  if (targetSize != null) {
    // If the target size is a pixel value, convert it to a percentage.
    if (targetSize > 1) targetSize = targetSize / prev.parentSize;
    diff = targetSize - prev.sizeDistribution[dragging];
  } else if (clientPos != null) {
    if (prev.root === null)
      throw new UnexpectedError("resize root is null during handle drag");
    diff = (clientPos - prev.root) / prev.parentSize;
  } else throw new Error("must provide either a MouseEvent or a targetSize");
  return diff;
};

export const distribute = (
  _state: ResizeMultipleState,
  count: number,
  minSize: number
): number[] => {
  if (_state.parentSize === null)
    throw new UnexpectedError("parent size is null during distribute");
  const { parentSize, sizeDistribution: state } = _state;
  let nextState = [...state];
  const arePercentages = nextState.every((size) => size <= 1);
  if (!arePercentages) nextState = nextState.map((size) => size / parentSize);

  const totalSize = nextState.reduce((a, b) => a + b, 0);

  // If we have fewer sizes than children, we need to approximate the
  // remaining sizes.
  if (nextState.length < count) {
    const diff = 1 - totalSize;
    const remaining = count - nextState.length;
    nextState = nextState.concat(
      Array.from({ length: remaining }, () => diff / remaining)
    );
  }

  if (totalSize < 0.99 || totalSize > 1.01)
    nextState = nextState.map((size) => size / totalSize);

  return nextState;
};

const resizeWithSibling = (
  prevDistribution: number[],
  item: number,
  diff: number,
  minSizePercent: number
): [number[], boolean] => {
  let next = item;

  // The while loops find a pane we can actually resize.
  while (next > 0 && prevDistribution[next] + diff <= minSizePercent) next--;
  const nextSize = prevDistribution[next] + diff;

  const nextSibling = findResizableSibling(
    next,
    prevDistribution,
    minSizePercent,
    diff
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
  diff: number
): number => {
  const f = item === sizes.length - 1 ? findBackward : findForward;
  return f(item, sizes, minSize, diff);
};

const findForward = (
  start: number,
  sizes: number[],
  minSize: number,
  diff: number
): number => {
  let i = start + 1;
  while (i < sizes.length - 1 && sizes[i] - diff <= minSize) i++;
  return i;
};

const findBackward = (
  start: number,
  sizes: number[],
  minSize: number,
  diff: number
): number => {
  let i = start - 1;
  while (i > 0 && sizes[i] - diff <= minSize) i--;
  return i;
};
