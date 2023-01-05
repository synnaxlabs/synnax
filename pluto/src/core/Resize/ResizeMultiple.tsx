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
  useEffect,
  useRef,
  useState,
  DragEvent as RDragEvent,
  MouseEvent as RMouseEvent,
} from "react";

import { ResizeCore } from "./ResizeCore";

import { Space, SpaceProps } from "@/core/Space";
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
  onDragHandle: (e: RDragEvent | RMouseEvent, i: number) => void;
}

export interface UseResizeMultipleReturn {
  setSize: (i: number, diff?: MouseEvent, size?: number) => void;
  props: ResizeMultipleProps & {
    ref: ForwardedRef<HTMLDivElement>;
  };
}

interface ResizeMultipleState {
  sizes: number[];
  root: number | null;
  dragging: number | null;
}

export const useResizeMultiple = ({
  count,
  onResize,
  initialSizes = [],
  minSize = 100,
  direction = "horizontal",
}: UseResizeMultipleProps): UseResizeMultipleReturn => {
  const [sizes, setSizes] = useState<ResizeMultipleState>({
    sizes: initialSizes,
    root: null,
    dragging: null,
  });
  const ref = useRef<HTMLDivElement>(null);

  const handleResize = useCallback(
    (dims: Dimensions) =>
      setSizes(({ sizes: prev, ...rest }) => {
        const f = (): number[] => {
          const nextPSize = getDirectionalSize(direction, dims);
          const prevPSize = prev.reduce((a, b) => a + b, 0);
          if (nextPSize === prevPSize || nextPSize === 0) return prev;

          // If the previous sizes aren't valid, simply distribute the space evenly
          // between all children.
          if (!validateSizes(count, prev))
            return Array.from({ length: count }, () => nextPSize / count);

          const sizePercentages = calculatePercentages(count, prev, nextPSize);
          return sizePercentages.map((size) => size * nextPSize);
        };
        return { sizes: f(), ...rest };
      }),
    [direction, count]
  );

  useResize({ ref, onResize: handleResize });

  const setSize = useCallback(
    (i: number, e?: MouseEvent, targetSize?: number) => {
      setSizes(({ sizes: prev, ...rest }) => {
        let diff: number;
        if (targetSize !== undefined) diff = targetSize - prev[i];
        else if (e != null) {
          if (rest.root === null) return { sizes: prev, ...rest };
          const curr = direction === "horizontal" ? e.clientX : e.clientY;
          diff = curr - rest.root;
        } else {
          throw new Error("Must provide either a MouseEvent or a targetSize");
        }

        const f = (): number[] | null => {
          const nextState = resizeWithSibling(prev, i, diff, minSize);
          if (nextState == null) return null;
          const { item, size, sibling, siblingSize } = nextState;

          const prevTotal = prev.reduce((a, b) => a + b, 0);
          const nextSizes = prev.map((prev, j) => {
            if (j === item) return size;
            if (j === sibling) return siblingSize;
            return prev;
          });
          const nextTotal = nextSizes.reduce((a, b) => a + b, 0);
          const r = nextSizes.map((s) => (s / nextTotal) * prevTotal);
          return r;
        };
        const nextSizes = f();
        if (nextSizes == null) return { sizes: prev, ...rest };
        const nxtRoot = direction === "horizontal" ? e?.clientX : e?.clientY;
        return { sizes: nextSizes, ...rest, root: nxtRoot as number };
      });
    },
    [minSize, setSizes, direction]
  );

  useEffect(() => {
    if (sizes.dragging === null) return;
    const onMouseMove = (e: MouseEvent): void => {
      if (sizes.dragging == null) return;
      setSize(sizes.dragging, e);
      onResize?.(sizes.sizes);
    };
    const onMouseUp = (): void => {
      setSizes(({ sizes }) => ({ sizes, root: null, marker: null, dragging: null }));
    };
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [sizes.dragging, onResize]);

  return {
    setSize,
    props: {
      direction,
      sizes: sizes.sizes,
      onDragHandle: (e, i) => {
        setSizes(({ sizes }) => {
          const root = direction === "horizontal" ? e.clientX : e.clientY;
          const marker = sizes[i];
          return {
            sizes,
            root,
            marker,
            dragging: i,
          };
        });
      },
      ref,
    },
  };
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
              onDragStart={(e) => onDrag(e, i)}
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
