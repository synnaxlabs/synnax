import clsx from "clsx";
import {
  ForwardedRef,
  forwardRef,
  HTMLAttributes,
  PropsWithChildren,
} from "react";
import { Direction } from "../../util/spatial";
import { ComponentSize } from "../../util/types";
import "./Space.css";

export type SpaceAlignment = "start" | "center" | "end" | "stretch";
export const SpaceAlignments = ["start", "center", "end", "stretch"] as const;
export type SpaceJustification =
  | "start"
  | "center"
  | "end"
  | "spaceBetween"
  | "spaceAround"
  | "spaceEvenly";
export const SpaceJustifications = [
  "start",
  "center",
  "end",
  "spaceBetween",
  "spaceAround",
  "spaceEvenly",
] as const;
export interface SpaceProps
  extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
  empty?: boolean;
  size?: ComponentSize | number | null;
  direction?: Direction;
  reverse?: boolean;
  justify?: SpaceJustification;
  align?: SpaceAlignment;
  grow?: boolean | number;
}

const juitifications = {
  start: "flex-start",
  center: "center",
  end: "flex-end",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};

const Space = forwardRef(
  (
    {
      empty = false,
      size = "medium",
      justify = "start",
      reverse = false,
      direction = "vertical",
      grow = false,
      align,
      className,
      style,
      children,
      ...props
    }: SpaceProps,
    ref: ForwardedRef<HTMLDivElement>
  ) => {
    let gap;
    if (empty) {
      size = null;
      gap = 0;
    } else if (typeof size == "string") {
      gap = `pluto-space--${size}`;
    } else {
      gap = `calc(var(--pluto-base-size) * ${size})`;
    }
    return (
      <div
        className={clsx(
          "pluto-space",
          typeof size === "string" ? "pluto-space--" + size : undefined,
          `pluto-space--${direction}`,
          className
        )}
        ref={ref}
        style={{
          flexDirection: flexDirection(direction, reverse),
          justifyContent: juitifications[justify],
          alignItems: align,
          flexGrow: Number(grow),
          gap,
          ...style,
        }}
        {...props}
      >
        {children}
      </div>
    );
  }
);
export default Space;

const flexDirection = (direction: Direction, reverse: boolean) => {
  if (direction === "horizontal") {
    return reverse ? "row-reverse" : "row";
  } else {
    return reverse ? "column-reverse" : "column";
  }
};
