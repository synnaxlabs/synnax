// Copyright 2022 Synnax Labs, Inc.
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
  useEffect,
  useRef,
  useState,
} from "react";

import { parseMovement } from "./Resize";
import { ResizeCore } from "./ResizeCore";

import { Space, SpaceProps } from "@/atoms/Space";
import { useResize } from "@/hooks";
import { Dimensions, Direction, getDirectionalSize, getLocation } from "@/util/spatial";

export interface UseResizeMultipleProps {
  count: number;
  onResize?: (sizes: number[]) => void;
  direction?: Direction;
  initialSizes?: number[];
  minSize?: number;
}

export interface ResizeMultipleProps extends SpaceProps {
  sizes: number[];
  onDragHandle: (i: number) => void;
}

export interface UseResizeMultipleReturn {
  setSize: (i: number, diff?: number, size?: number) => void;
  props: ResizeMultipleProps & {
    ref: ForwardedRef<HTMLDivElement>;
  };
}

export const useResizeMultiple = ({
  count,
  onResize,
  initialSizes = [],
  minSize = 100,
  direction = "horizontal",
}: UseResizeMultipleProps): UseResizeMultipleReturn => {
  const [sizes, setSizes] = useState<number[]>(initialSizes);
  const ref = useRef<HTMLDivElement>(null);
  const location = getLocation(direction);
  const [dragging, setDragging] = useState<number | undefined>(undefined);

  const handleResize = useCallback(
    (dims: Dimensions) =>
      setSizes((prevSizes) => {
        const nextPSize = getDirectionalSize(direction, dims);
        const prevPSize = prevSizes.reduce((a, b) => a + b, 0);
        if (nextPSize === prevPSize || nextPSize === 0) return prevSizes;

        // If the previous sizes aren't valid, simply distribute the space evenly
        // between all children.
        if (!validateSizes(count, prevSizes))
          return Array.from({ length: count }, () => nextPSize / count);

        const sizePercentages = calculatePercentages(count, prevSizes, nextPSize);
        return sizePercentages.map((size) => size * nextPSize);
      }),
    [direction, count]
  );

  useResize({ ref, onResize: handleResize });

  const setSize = useCallback(
    (i: number, diff?: number, targetSize?: number) => {
      setSizes((prevSizes: number[]) => {
        diff = diff ?? 0;
        if (targetSize !== undefined) diff = targetSize - prevSizes[i];
        const nextState = resizeWithSibling(prevSizes, i, diff, minSize);
        if (nextState == null) return prevSizes;
        const { item, size, sibling, siblingSize } = nextState;

        const prevTotal = prevSizes.reduce((a, b) => a + b, 0);
        const nextSizes = prevSizes.map((prev, j) => {
          if (j === item) return size;
          if (j === sibling) return siblingSize;
          return prev;
        });
        const nextTotal = nextSizes.reduce((a, b) => a + b, 0);
        const r = nextSizes.map((s) => (s / nextTotal) * prevTotal);
        return r;
      });
    },
    [minSize, setSizes]
  );

  useEffect(() => {
    if (dragging === undefined) return;
    const onMouseMove = (e: MouseEvent): void => {
      const movement = parseMovement(location, e);
      setSize(dragging, movement);
      const totalSize = sizes.reduce((a, b) => a + b, 0);
      onResize?.(sizes.map((size) => size / totalSize));
    };
    const onMouseUp = (): void => setDragging(undefined);
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [dragging, onResize]);

  return { setSize, props: { direction, sizes, onDragHandle: setDragging, ref } };
};

export const ResizeMultiple = forwardRef(
  (
    {
      direction = "horizontal",
      children: _children,
      sizes,
      onDragHandle: onDrag,
      ...props
    }: ResizeMultipleProps,
    ref: ForwardedRef<HTMLDivElement>
  ) => {
    const children = Children.toArray(_children);
    const location = getLocation(direction);

    return (
      <Space
        ref={ref}
        className="pluto-multi-resizable__container"
        direction={direction}
        empty
        grow
        {...props}
      >
        {children.map((child, i) => {
          return (
            <ResizeCore
              onDragStart={() => onDrag(i)}
              key={i}
              location={location}
              size={sizes[i]}
              showHandle={i !== children.length - 1}
            >
              {child}
            </ResizeCore>
          );
        })}
      </Space>
    );
  }
);
ResizeMultiple.displayName = "ResizeMultiple";

const validateSizes = (numChildren: number, sizes: number[]): boolean =>
  sizes.length >= numChildren - 1;

const calculatePercentages = (
  numChildren: number,
  sizes: number[],
  parentSize: number
): number[] => {
  const arePercentages = sizes.every((size) => size <= 1);
  if (!arePercentages) sizes = sizes.map((size) => size / parentSize);

  let totalSize = sizes.reduce((a, b) => a + b, 0);

  // If we have fewer sizes than children, we need to approximate the
  // remaining sizes.
  if (sizes.length < numChildren) {
    const diff = 1 - totalSize;
    const remaining = numChildren - sizes.length;
    sizes = sizes.concat(Array.from({ length: remaining }, () => diff / remaining));
  }

  totalSize = sizes.reduce((a, b) => a + b, 0);

  // If our total size is not equal to 1, we need to scale our sizes. Explicitly do a
  // tolerance check to avoid floating point errors.
  if (totalSize < 0.99 || totalSize > 1.01)
    sizes = sizes.map((size) => size / totalSize);

  return sizes;
};

const resizeWithSibling = (
  prevSizes: number[],
  item: number,
  diff: number,
  minSize: number
): {
  item: number;
  size: number;
  sibling: number;
  siblingSize: number;
} | null => {
  let next = item;

  // The while loops find a pane we can actually resize.
  while (next > 0 && prevSizes[next] + diff <= minSize) next--;
  const nextSize = prevSizes[next] + diff;

  const nextSibling = findResizableSibling(next, prevSizes, minSize, diff);
  const nextSiblingSize = prevSizes[nextSibling] - diff;

  // This means we can't resize any panes so we return null to indicate that the next
  // set of sizes should remain the same.
  if (nextSize < minSize || nextSiblingSize < minSize) return null;

  return {
    item: next,
    size: nextSize,
    sibling: nextSibling,
    siblingSize: nextSiblingSize,
  };
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
  while (i > 0 && sizes[i] + diff <= minSize) i--;
  return i;
};
