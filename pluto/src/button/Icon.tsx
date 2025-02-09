// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon as MediaIcon } from "@synnaxlabs/media";
import clsx from "clsx";
import { cloneElement, type ReactElement, useCallback } from "react";

import { type BaseProps } from "@/button/Button";
import { color } from "@/button/color";
import { CSS } from "@/css";
import { Tooltip } from "@/tooltip";

interface ChildProps {
  color?: string;
  fill?: string;
}

/** The props for the {@link Icon} component */
export interface IconProps extends BaseProps, Tooltip.WrapProps {
  children: ReactElement<ChildProps> | string;
  loading?: boolean;
}

/**
 * Use.Icon a button that only renders an icon without any text.
 *
 * @param props - Props for the component, which are passed down to the underlying
 * element.
 * @param props.size - The size of button to render.
 * @param props.variant - The variant of button to render. Options are "filled" (default),
 * "outlined", and "text".
 * @param props.children - A ReactElement representing the icon to render.
 * @param props.loading - Whether the button is in a loading state. This will cause the
 * button to render a loading spinner.
 */
export const Icon = Tooltip.wrap(
  ({
    ref,
    children,
    stopPropagation,
    className,
    variant = "text",
    size = "medium",
    sharp = false,
    disabled = false,
    loading = false,
    onClick,
    color: propColor,
    ...props
  }: IconProps): ReactElement => {
    if (loading) children = <MediaIcon.Loading />;
    const isDisabled = disabled || loading;
    const handleClick = useCallback(
      (e: React.MouseEvent<HTMLButtonElement>) => {
        if (stopPropagation) e.stopPropagation();
        if (isDisabled) return;
        onClick?.(e);
      },
      [stopPropagation, isDisabled, onClick],
    );
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
          CSS.disabled(isDisabled),
        )}
        onClick={handleClick}
        {...props}
      >
        {typeof children === "string"
          ? children
          : cloneElement(children, {
              color: color(variant, isDisabled, propColor),
              fill: "currentColor",
              ...children.props,
            })}
      </button>
    );
  },
);
