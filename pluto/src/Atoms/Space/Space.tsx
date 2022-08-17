import { HTMLAttributes, PropsWithChildren } from "react";
import { classList } from "../../util/css";
import "./Space.css";

export interface SpaceProps
  extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
  empty?: boolean;
  size?: "small" | "medium" | "large" | number;
  direction?: "horizontal" | "vertical";
  justify?:
    | "start"
    | "end"
    | "center"
    | "spaceBetween"
    | "spaceAround"
    | "spaceEvenly";
  align?: "start" | "end" | "center" | "stretch";
}

const justifyMap = {
  start: "flex-start",
  end: "flex-end",
  center: "center",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};

const Space = ({
  empty = false,
  size = "medium",
  justify = "start",
  children,
  align,
  ...props
}: SpaceProps) => {
  let gap;
  if (empty) {
    gap = 0
  } else if (typeof size == "string")  {
    gap = `pluto-space--${size}`
  } else{
    gap = `calc(var(--base-size) * ${size})`
  }
  return (
    <div
      className={classList(
        "pluto-space",
        typeof size === "string" ? "pluto-space--" + size : undefined,
        props.className
      )}
      style={{
        flexDirection: props.direction === "horizontal" ? "row" : "column",
        gap,
        justifyContent: justify && justifyMap[justify],
        alignItems: align,
        ...props.style,
      }}
    >
      {children}
    </div>
  );
}

export default Space;