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
  useEffect,
  useRef,
  RefObject,
} from "react";

import { Box, ClientXY, Direction, locFromDir } from "@synnaxlabs/x";

import { ResizeCore } from "@/core/Resize/ResizeCore";
import "@/core/Resize/ResizeMultiple.css";
import { Space, SpaceProps } from "@/core/Space";
import { CSS } from "@/css";

/** Props for the {@link Resize.Multiple} component. */
export interface ResizeMultipleProps extends SpaceProps {
  sizeDistribution: number[];
  onDragHandle: (e: ResizeStartEvent, i: number) => void;
}

export type ResizeStartEvent = ClientXY & { preventDefault: () => void };

/** Props for the {@link Resize.useMultiple} hook. */
export interface UseResizeMultipleProps {
  count: number;
  onResize?: (sizes: number[]) => void;
  direction?: Direction;
  initialSizes?: number[];
  minSize?: number;
}

/** Return value of the {@link Resize.useMultiple} hook. */
export interface UseResizeMultipleReturn {
  setSize: (i: number, size?: number) => void;
  props: Pick<
    ResizeMultipleProps,
    "sizeDistribution" | "onDragHandle" | "direction"
  > & {
    ref: RefObject<HTMLDivElement>;
  };
}

interface ResizeMultipleState {
  sizeDistribution: number[];
  root: number | null;
}

export const useResizeMultiple = ({
  onResize,
  count,
  initialSizes = [],
  minSize = 100,
  direction = "x",
}: UseResizeMultipleProps): UseResizeMultipleReturn => {
  const ref = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<ResizeMultipleState>({
    sizeDistribution: calculateInitialSizeDistribution(initialSizes, count),
    root: null,
  });

  const _handleResize = useCallback(
    (dragging: number, clientPos?: number, targetSize?: number) => {
      if (ref.current == null) return;
      const parentSize = new Box(ref.current).dim(direction);
      setState((prev) =>
        handleResize(prev, parentSize, dragging, minSize, clientPos, targetSize)
      );
    },
    [minSize, setState, direction]
  );

  useEffect(
    () => onResize?.(state.sizeDistribution),
    [state.sizeDistribution, onResize]
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
    [setState, direction, handleResize]
  );

  const setSize = useCallback(
    (i: number, size?: number) => _handleResize(i, undefined, size),
    [handleResize]
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

export const ResizeMultiple = forwardRef(
  (
    {
      direction = "x",
      children: _children,
      sizeDistribution,
      onDragHandle: onDrag,
      className,
      ...props
    }: ResizeMultipleProps,
    ref: ForwardedRef<HTMLDivElement>
  ) => {
    const children = Children.toArray(_children);
    const location = locFromDir(direction);

    return (
      <Space
        {...props}
        ref={ref}
        direction={direction}
        className={CSS(CSS.B("resize-multiple"), className)}
        empty
        grow
      >
        {children.map((child, i) => (
          <ResizeCore
            onDragStart={(e) => onDrag(e, i)}
            key={i}
            location={location}
            size={sizeDistribution[i] * 100}
            sizeUnits="%"
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

const calculateInitialSizeDistribution = (
  initial: number[],
  count: number
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

const handleResize = (
  prev: ResizeMultipleState,
  parentSize: number,
  dragging: number,
  minSize: number,
  clientPos?: number,
  targetSize?: number
): ResizeMultipleState => {
  const diffPercentage = calculateDiffPercentage(
    prev,
    parentSize,
    dragging,
    clientPos,
    targetSize
  );
  const [sizeDistribution, changed] = resizeWithSibling(
    prev.sizeDistribution,
    dragging,
    diffPercentage,
    minSize / parentSize
  );
  const root = changed ? clientPos ?? null : prev.root;
  return { ...prev, sizeDistribution, root };
};

export const calculateDiffPercentage = (
  prev: ResizeMultipleState,
  parentSize: number,
  dragging: number,
  clientPos?: number,
  targetSize?: number
): number => {
  let diff: number;
  // If the caller provided a target size, prefer that.
  if (targetSize != null) {
    // If the target size is a pixel value, convert it to a percentage.
    if (targetSize > 1) targetSize = targetSize / parentSize;
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
