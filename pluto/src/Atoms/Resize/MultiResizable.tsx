import { Children, HTMLAttributes } from "react";
import { Direction, getLocation } from "../../util/spatial";
import Space from "../Space/Space";
import ResizePanel from "./ResizePanel";

export interface MultiResizableProps extends HTMLAttributes<HTMLDivElement> {
  direction: Direction;
}

export default function MultiResizable({
  children,
  direction,
}: MultiResizableProps) {
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
          <ResizePanel
            key={i}
            location={getLocation(direction)}
            style={{ zIndex: i, overflow: "hidden" }}
          >
            {child}
          </ResizePanel>
        );
      })}
      <div style={{ flexGrow: 1, overflow: "hidden" }}>
        {childArray[childArray.length - 1]}
      </div>
    </Space>
  );
}
