import { Children, HTMLAttributes } from "react";
import { Direction, getLocation } from "../../util/spatial";
import Space from "../Space/Space";
import Resize from "./Resize";

export interface ResizeMultipleProps extends HTMLAttributes<HTMLDivElement> {
  direction: Direction;
}

export default function ResizeMultiple({
  children,
  direction,
}: ResizeMultipleProps) {
  const childArray = Children.toArray(children);
  return (
    <Space
      className="pluto-multi-resizable__container"
      direction={direction}
      style={{
        height: "100%",
        flexBasis: "0%",
        flexGrow: 0,
      }}
      empty
      grow
    >
      {childArray.slice(0, childArray.length - 1).map((child, i) => {
        return (
          <Resize
            key={i}
            location={getLocation(direction)}
            style={{ zIndex: i, overflow: "hidden" }}
          >
            {child}
          </Resize>
        );
      })}
      <div style={{ flexGrow: 1, overflow: "hidden" }}>
        {childArray[childArray.length - 1]}
      </div>
    </Space>
  );
}
