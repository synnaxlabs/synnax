import { Children, useCallback, useEffect, useRef, useState } from "react";
import { useResize } from "@/hooks";
import { Dimensions, getDirectionalSize, getLocation } from "@/util";
import { Space, SpaceProps } from "@/atoms/Space";
import { anyExceedsBounds, parseMovement } from "./Resize";
import { ResizeCore } from "./ResizeCore";

export interface ResizeMultipleProps extends SpaceProps {
  onResize?: (sizes: number[]) => void;
  initialSizes?: number[];
  maxSize?: number;
  minSize?: number;
}

export const ResizeMultiple = ({
  direction = "horizontal",
  onResize,
  initialSizes = [],
  children: _children,
  maxSize = Infinity,
  minSize = 100,
  ...props
}: ResizeMultipleProps) => {
  const children = Children.toArray(_children);

  const [sizes, setSizes] = useState<number[]>(initialSizes);
  const parentRef = useRef<HTMLDivElement>(null);
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
        if (!validateSizes(children.length, prevSizes))
          return Array.from(
            { length: children.length },
            () => nextPSize / children.length
          );

        let sizePercentages = calculatePercentages(
          children.length,
          prevSizes,
          nextPSize
        );
        return sizePercentages.map((size) => size * nextPSize);
      }),
    [direction, children.length]
  );

  useResize({ ref: parentRef, onResize: handleResize });

  useEffect(() => {
    if (dragging === undefined) return;
    const onMouseMove = (e: MouseEvent) => {
      setSizes((prevSizes: number[]) => {
        const movement = parseMovement(location, e);
        const next = prevSizes[dragging] + movement;
        const nextSibling = prevSizes[dragging + 1] - movement;
        if (anyExceedsBounds([next, nextSibling], minSize, maxSize))
          return prevSizes;
        return prevSizes.map((size, i) => {
          if (i === dragging) return next;
          if (i === dragging + 1) return nextSibling;
          return size;
        });
      });
      const totalSize = sizes.reduce((a, b) => a + b, 0);
      onResize?.(sizes.map((size) => size / totalSize));
    };
    const onMouseUp = () => setDragging(undefined);
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [dragging, onResize]);

  return (
    <Space
      ref={parentRef}
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
};

const validateSizes = (numChildren: number, sizes: number[]): boolean =>
  sizes.length >= numChildren - 1;

const calculatePercentages = (
  numChildren: number,
  sizes: number[],
  parentSize: number
) => {
  const arePercentages = sizes.every((size) => size <= 1);
  if (!arePercentages) {
    sizes = sizes.map((size) => size / parentSize);
  }

  let totalSize = sizes.reduce((a, b) => a + b, 0);

  // If we have fewer sizes than children, we need to approximate the
  // remaining sizes.
  if (sizes.length < numChildren) {
    const diff = 1 - totalSize;
    const remaining = numChildren - sizes.length;
    sizes = sizes.concat(
      Array.from({ length: remaining }, () => diff / remaining)
    );
  }

  totalSize = sizes.reduce((a, b) => a + b, 0);

  // If our total size is not equal to 1, we need to scale our sizes
  if (totalSize < 0.99 || totalSize > 1.01) {
    sizes = sizes.map((size) => size / totalSize);
  }

  return sizes;
};
