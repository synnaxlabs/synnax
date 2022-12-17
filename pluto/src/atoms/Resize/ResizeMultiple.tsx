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
import { Dimensions, Direction, getDirectionalSize, getLocation } from "@/util";

export interface UseResizeMultipleProps {
  itemCount: number;
  onResize?: (sizes: number[]) => void;
  direction?: Direction;
  initialSizes?: number[];
  maxSize?: number;
  minSize?: number;
}

export interface ResizeMultipleProps extends SpaceProps {
  sizes: number[];
  setDragging: (i: number) => void;
  setSize: (i: number, diff?: number, size?: number) => void;
}

export interface UseResizeMultipleReturn {
  direction: Direction;
  sizes: number[];
  setDragging: (i: number) => void;
  setSize: (i: number, diff?: number, size?: number) => void;
  ref: ForwardedRef<HTMLDivElement>;
}

export const useResizeMultiple = ({
  itemCount,
  onResize,
  initialSizes = [],
  maxSize = Infinity,
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
        if (!validateSizes(itemCount, prevSizes))
          return Array.from({ length: itemCount }, () => nextPSize / itemCount);

        const sizePercentages = calculatePercentages(itemCount, prevSizes, nextPSize);
        return sizePercentages.map((size) => size * nextPSize);
      }),
    [direction, itemCount]
  );

  useResize({ ref, onResize: handleResize });

  const setSize = useCallback(
    (i: number, diff?: number, size?: number) => {
      setSizes((prevSizes: number[]) => {
        diff = diff ?? 0;
        if (size !== undefined) diff = size - prevSizes[i];

        let next = prevSizes[i] + diff;
        const nextSiblingIndex = i + 1 < prevSizes.length ? i + 1 : i - 1;
        let nextSibling = prevSizes[nextSiblingIndex] - diff;
        if (next < minSize) {
          next = minSize;
          nextSibling = prevSizes[nextSiblingIndex] - (minSize - prevSizes[i]);
        } else if (nextSibling < minSize) {
          nextSibling = minSize;
          next = prevSizes[i] - (minSize - prevSizes[nextSiblingIndex]);
        }

        return prevSizes.map((size, j) => {
          if (j === i) return next;
          if (j === nextSiblingIndex) return nextSibling;
          return size;
        });
      });
    },
    [maxSize, minSize, setSizes]
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

  return { direction, sizes, setDragging, setSize, ref };
};

export const ResizeMultiple = forwardRef(
  (
    {
      direction = "horizontal",
      children: _children,
      sizes,
      setDragging,
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
              onDragStart={() => setDragging(i)}
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
