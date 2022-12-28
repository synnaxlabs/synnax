import { Children, cloneElement, ReactElement } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "../Space";

import { ComponentSize } from "@/util/component";

import "./Group.css";

export interface GroupProps
  extends Omit<SpaceProps, "children" | "empty" | "direction"> {
  children: ReactElement | ReactElement[];
  size?: ComponentSize;
}

export const Group = ({
  children,
  size = "medium",
  ...props
}: GroupProps): JSX.Element => {
  const arr = Children.toArray(children) as ReactElement[];
  return (
    <Space {...props} direction="horizontal" empty>
      {arr.map((child, index) =>
        cloneElement(child, {
          key: index,
          className: clsx(
            "pluto-group__item",
            groupClassName(index, arr.length),
            `pluto--${size}`,
            child.props.className
          ),
          size,
        })
      )}
    </Space>
  );
};

const groupClassName = (i: number, length: number): string => {
  if (i === 0 && i === length - 1) return "pluto-group__only";
  if (i === 0) return "pluto-group__first";
  if (i === length - 1) return "pluto-group__last";
  return "pluto-group__middle";
};
