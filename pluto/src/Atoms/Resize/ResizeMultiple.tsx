import clsx from "clsx";
import { drag } from "d3";
import {
  Children,
  HTMLAttributes,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { useResize } from "../../Hooks";
import {
  Direction,
  getDirection,
  getLocation,
  swapLocation,
} from "../../util/spatial";
import { AutoSize } from "../AutoSize";
import Space from "../Space/Space";
import Resize, { calcNextSize, ResizePanelProps } from "./Resize";

export interface ResizeMultipleProps extends HTMLAttributes<HTMLDivElement> {
  direction: Direction;
  onResize?: (sizes: number[]) => void;
  initialSizes?: number[];
}

export default function ResizeMultiple({
  children,
  direction,
  onResize,
  initialSizes = [],
  ...props
}: ResizeMultipleProps) {
  const childArray = Children.toArray(children);
  const [sizes, setSizes] = useState<number[]>(initialSizes);
  const parentRef = useRef<HTMLDivElement>(null);
  const location = getLocation(direction);

  const [dragging, setDragging] = useState<number | undefined>(undefined);

  const { width, height } = useResize(parentRef);

  useEffect(() => {
    // distribute the size of the parent ref equally among the children
    const parentSize = direction === "horizontal" ? width : height;
    const childSize = parentSize / childArray.length;
    setSizes(Array(childArray.length).fill(childSize));
  }, [width, height]);

  useEffect(() => {
    if (dragging === undefined) return;
    const onMouseMove = (e: MouseEvent) => {
      setSizes((prevSizes: number[]) => {
        console.log(dragging);
        const next = calcNextSize(e, location, prevSizes[dragging], 200, 80000);
        // calc diff
        const diff = next - prevSizes[dragging];
        // add or subtract diff evenly to all other sizes
        const newSizes = prevSizes.map((size, i) => {
          if (i === dragging) return next;
          return size - diff / (prevSizes.length - 1);
        });
        return newSizes;
      });
    };
    const onMouseUp = () => setDragging(undefined);
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [dragging, onResize]);

  const _onResize = useCallback(
    (i: number, size: number) => {
      setSizes((prevSizes) => {
        const nextSizes = [...prevSizes];
        nextSizes[i] = size;
        return nextSizes;
      });
      onResize?.(sizes);
    },
    [onResize]
  );
  return (
    <Space
      ref={parentRef}
      className="pluto-multi-resizable__container"
      direction={direction}
      empty
      grow
      {...props}
    >
      {childArray.map((child, i) => {
        return (
          <BaseResize
            onDrag={(i: number) => setDragging(i)}
            key={i}
            index={i}
            location={location}
            size={sizes[i]}
            showHandle={i !== childArray.length - 1}
          >
            {child}
          </BaseResize>
        );
      })}
    </Space>
  );
}

const ResizeChild = ({
  onResize,
  index,
  initialSizes,
  ...props
}: Omit<ResizePanelProps, "onResize" | "initialSize"> & {
  index: number;
  onResize: (index: number, size: number) => void;
  initialSizes: number[];
}) => {
  const initialSize = initialSizes[index];
  const _onResize = useCallback(
    (size: number) => onResize(index, size),
    [onResize]
  );
  return <Resize {...props} onResize={_onResize} />;
};

const BaseResize = ({
  location,
  style,
  size,
  className,
  children,
  onDrag,
  index,
  showHandle,
  ...props
}: Omit<ResizePanelProps, "onDrag"> & {
  index: number;
  size: number;
  showHandle: boolean;
  onDrag: (i: number) => void;
}) => {
  const direction = getDirection(location);
  const parsedStyle: React.CSSProperties = { ...style, overflow: "hidden" };
  if (direction === "horizontal") {
    parsedStyle.height = size;
  } else {
    parsedStyle.width = size;
  }
  return (
    <div
      className={clsx(
        "pluto-resize-panel",
        `pluto-resize-panel--${location}`,
        `pluto-resize-panel--${direction}`,
        `pluto-bordered--${swapLocation(location)}`,
        className
      )}
      style={parsedStyle}
      {...props}
    >
      {children}
      {showHandle && (
        <div
          draggable
          className="pluto-resize-panel__handle"
          data-testid="resize-handle"
          onDragStart={(e) => {
            e.preventDefault();
            onDrag(index);
          }}
          onDrag={(e) => e.preventDefault()}
          onDragEnd={(e) => e.preventDefault()}
        ></div>
      )}
    </div>
  );
};
