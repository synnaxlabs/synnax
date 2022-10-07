import clsx from "clsx";
import {
  forwardRef,
  HTMLAttributes,
  PropsWithChildren,
  RefObject,
} from "react";
import "./Space.css";

type SpaceDirection = "horizontal" | "vertical";

export interface SpaceProps
  extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
  empty?: boolean;
  size?: "small" | "medium" | "large" | number;
  direction?: SpaceDirection;
  reverse?: boolean;
  justify?:
    | "start"
    | "end"
    | "center"
    | "spaceBetween"
    | "spaceAround"
    | "spaceEvenly";
  align?: "start" | "end" | "center" | "stretch";
  grow?: boolean | number;
}

const justifyMap = {
  start: "flex-start",
  end: "flex-end",
  center: "center",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};

const Space = forwardRef<HTMLDivElement, SpaceProps>(
  (
    {
      empty = false,
      size = "medium",
      justify = "start",
      reverse = false,
      direction = "vertical",
      children,
      align,
      ...props
    }: SpaceProps,
    ref
  ) => {
    let gap;
    if (empty) {
      size = "";
      gap = 0;
    } else if (typeof size == "string") {
      gap = `pluto-space--${size}`;
    } else {
      gap = `calc(var(--pluto-base-size) * ${size})`;
    }
    if (props.grow === true) {
      props.grow = 1;
    }
    return (
      <div
        {...props}
        className={clsx(
          "pluto-space",
          typeof size === "string" ? "pluto-space--" + size : undefined,
          props.className
        )}
        ref={ref}
        style={{
          flexDirection: getDirection(direction, reverse),
          gap,
          justifyContent: justifyMap[justify],
          alignItems: align,
          flexGrow: props.grow ? props.grow : undefined,
          ...props.style,
        }}
      >
        {children}
      </div>
    );
  }
);

export default Space;

const getDirection = (direction: SpaceDirection, reverse: boolean) => {
  if (direction === "horizontal") {
    return reverse ? "row-reverse" : "row";
  } else {
    return reverse ? "column-reverse" : "column";
  }
};
