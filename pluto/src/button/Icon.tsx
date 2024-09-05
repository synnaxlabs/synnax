// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon as MediaIcon } from "@synnaxlabs/media";
import clsx from "clsx";
import { cloneElement, forwardRef, type ReactElement } from "react";

import type { BaseProps } from "@/button/Button";
import { color } from "@/button/color";
import { CSS } from "@/css";
import { Tooltip } from "@/tooltip";

interface ChildProps {
  color?: string;
  fill?: string;
}

/** The props for the {@link Icon} */
export interface IconProps extends BaseProps {
  children: ReactElement<ChildProps> | string;
  loading?: boolean;
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
      loading = false,
      onClick,
      color: propColor,
      ...props
    },
    ref,
  ): ReactElement => {
    if (loading) children = <MediaIcon.Loading />;
    return (
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
        {typeof children === "string"
          ? children
          : cloneElement(children, {
              color: color(variant, disabled, propColor),
              fill: "currentColor",
              ...children.props,
            })}
      </button>
    );
  },
);

CoreIcon.displayName = "ButtonIcon";

/**
 * Use.Icon a button that only renders an icon without any text.
 *
 * @param props - Props for the component, which are passed down to the underlying
 * element.
 * @param props.size - The size of button to render.
 * @param props.variant - The variant of button to render. Options are "filled" (default),
 * "outlined", and "text".
 * @param props.children - A ReactElement representing the icon to render.
 */
export const Icon = Tooltip.wrap(CoreIcon);
