// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cloneElement, ReactElement } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "@/core/Space";
import { reactElementToArray } from "@/util/children";
import { ComponentSize } from "@/util/component";

import "./Pack.css";

export interface PackChildProps {
  key: number;
  className: string;
  size: ComponentSize;
}

export interface PackProps extends Omit<SpaceProps, "children" | "empty"> {
  children: ReactElement<PackChildProps> | Array<ReactElement<PackChildProps>>;
  size?: ComponentSize;
}

/**
 * Packs elements together, setting their size and styling the borders between them so
 * that they appear as a single element. This is useful for buttons that represent a
 * selection state, for example.
 *
 * @param props - The props for the pack. Any extra props will be passed to the
 * underlying Space component.
 * @param props.children - The children to pack together. These must satisfy the
 * {@link PackChildProps} interface.
 * @param props.direction - The direction to pack the children in. Defaults to
 * "horizontal".
 * @param props.size - The size to set on the children. Any sizes already set on the
 * children will be overridden. Defaults to "medium".
 */
export const Pack = ({
  children,
  size = "medium",
  className,
  direction = "horizontal",
  ...props
}: PackProps): JSX.Element => {
  const arr = reactElementToArray(children);
  return (
    <Space
      direction={direction}
      className={clsx(`pluto-pack--${direction}`, className)}
      {...props}
      empty
    >
      {reactElementToArray(arr).map((child, index) =>
        cloneElement(child, {
          // using index as key is safe here because the children are unlikely to change
          // order.
          key: index,
          className: clsx(
            "pluto-pack__item",
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
  const [first, last] = [i === 0, i === length - 1];
  if (first && last) return "pluto-pack__only";
  if (first) return "pluto-pack__first";
  if (last) return "pluto-pack__last";
  return "pluto-pack__middle";
};
