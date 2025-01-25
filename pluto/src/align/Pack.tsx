// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/align/Pack.css";

import { type ReactElement } from "react";

import { Space, type SpaceElementType, type SpaceProps } from "@/align/Space";
import { CSS } from "@/css";
import { type text } from "@/text/core";

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
 * @param props.el  - The element type to use as the root element for the Pack.
 * Defaults to "div".
 */
export type PackProps<E extends SpaceElementType = "div"> = Omit<
  SpaceProps<E>,
  "empty"
> & {
  shadow?: boolean;
  borderWidth?: number;
};

export const Pack = <E extends SpaceElementType = "div">({
  className,
  size = "medium",
  reverse = false,
  direction = "x",
  bordered = true,
  borderShade = 3 as text.Shade,
  rounded = true,
  shadow = false,
  borderWidth,
  style,
  ...props
}: PackProps<E>): ReactElement => {
  const pStyle = {
    [CSS.var("pack-border-shade")]: CSS.shadeVar(borderShade),
    ...style,
  };
  if (borderWidth != null)
    // @ts-expect-error - generic element issues
    pStyle[CSS.var("pack-border-width")] = `${borderWidth}px`;

  return (
    // @ts-expect-error - generic element issues
    <Space<E>
      direction={direction}
      reverse={reverse}
      className={CSS(
        CSS.B("pack"),
        shadow && CSS.BM("pack", "shadow"),
        CSS.dir(direction),
        typeof size !== "number" && CSS.BM("pack", size),
        reverse && CSS.BM("pack", "reverse"),
        className,
      )}
      style={pStyle}
      bordered={bordered}
      rounded={rounded}
      {...props}
      empty
    />
  );
};
