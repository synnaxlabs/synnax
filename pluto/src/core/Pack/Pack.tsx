// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ForwardedRef, forwardRef, ReactElement } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "@/core/Space";
import { ComponentSize } from "@/util/component";

import "./Pack.css";

export interface PackChildProps {
  key: number;
  className: string;
  size: ComponentSize;
}

export interface PackProps<E extends HTMLElement = HTMLDivElement>
  extends Omit<SpaceProps<E>, "children" | "empty"> {
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
 * "x".
 * @param props.size - The size to set on the children. Any sizes already set on the
 * children will be overridden. Defaults to "medium".
 */
const CorePack = <E extends HTMLElement = HTMLDivElement>(
  {
    children,
    className,
    size = "medium",
    reverse = false,
    direction = "x",
    ...props
  }: PackProps<E>,
  ref: ForwardedRef<E>
): JSX.Element => (
  <Space<E>
    ref={ref}
    direction={direction}
    reverse={reverse}
    className={clsx(
      "pluto-pack",
      `pluto-pack--${direction}`,
      `pluto-pack--${size}`,
      reverse && "pluto-pack--reverse",
      className
    )}
    {...props}
    empty
  >
    {children}
  </Space>
);

export const Pack = forwardRef(CorePack) as <E extends HTMLElement = HTMLDivElement>(
  props: PackProps<E> & { ref?: ForwardedRef<E> }
) => JSX.Element;
