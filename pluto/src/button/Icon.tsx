// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cloneElement, forwardRef, type ReactElement } from "react";

import clsx from "clsx";

import type { BaseProps } from "@/button/Button";
import { CSS } from "@/css";
import { Tooltip } from "@/tooltip";

/** The props for the {@link Icon} */
export interface IconProps extends BaseProps {
  children: ReactElement;
}

const CoreIcon = forwardRef<HTMLButtonElement, IconProps>(
  (
    {
      children,
      className,
      variant = "text",
      size = "medium",
      sharp = false,
      disabled = false,
      onClick,
      ...props
    },
    ref,
  ): ReactElement => (
    <button
      ref={ref}
      className={clsx(
        className,
        CSS.B("btn"),
        CSS.B("btn-icon"),
        CSS.size(size),
        CSS.sharp(sharp),
        CSS.BM("btn", variant),
        CSS.disabled(disabled),
      )}
      onClick={disabled ? undefined : onClick}
      {...props}
    >
      {cloneElement(children, {
        color: color_(disabled, props.color),
        ...children.props,
      })}
    </button>
  ),
);
CoreIcon.displayName = "ButtonIcon";

const color_ = (disabled?: boolean, color?: string): string => {
  if (disabled === true) return "var(--pluto-gray-m2)";
  if (color != null) return color;
  return "var(--pluto-text-color)";
};

/**
 * Button.Icon a button that only renders an icon without any text.
 *
 * @param props - Props for the component, which are passed down to the underlying
 * element.
 * @param props.size - The size of button to render.
 * @param props.variant - The variant of button to render. Options are "filled" (default),
 * "outlined", and "text".
 * @param props.children - A ReactElement representing the icon to render.
 */
export const Icon = Tooltip.wrap(CoreIcon);
