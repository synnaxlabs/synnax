// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";
import { cloneElement, type ReactElement } from "react";

import { type BaseProps } from "@/button/Button";
import { parseColor } from "@/button/color";
import { CSS } from "@/css";
import { Icon as BaseIcon } from "@/icon";
import { type Text } from "@/text";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

interface ChildProps {
  color?: string;
  fill?: string;
}

/** The props for the {@link Icon} component */
export interface IconProps extends BaseProps, Tooltip.WrapProps {
  children: ReactElement<ChildProps> | string;
  shade?: Text.Shade;
  triggerIndicator?: Triggers.Trigger;
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
    className,
    variant = "text",
    size = "medium",
    sharp = false,
    disabled = false,
    loading = false,
    onClick,
    shade = 0,
    color: propColor,
    tabIndex,
    onMouseDown,
    triggerIndicator,
    ...rest
  }: IconProps): ReactElement => {
    if (loading) children = <BaseIcon.Loading />;
    const isDisabled = disabled || loading;
    return (
      <button
        ref={ref}
        className={clsx(
          className,
          CSS.B("btn"),
          CSS.M("clickable"),
          CSS.B("btn-icon"),
          CSS.size(size),
          CSS.sharp(sharp),
          CSS.M(variant),
          CSS.disabled(isDisabled),
          CSS.shade(shade),
        )}
        onClick={isDisabled ? undefined : onClick}
        onMouseDown={(e) => {
          if (tabIndex == -1) e.preventDefault();
          onMouseDown?.(e);
        }}
        tabIndex={tabIndex}
        {...rest}
      >
        {triggerIndicator && (
          <div className={CSS.B("trigger-indicator")}>
            <Triggers.Text trigger={triggerIndicator} level="small" />
          </div>
        )}
        {typeof children === "string"
          ? children
          : cloneElement(children, {
              color: parseColor(variant, isDisabled, propColor),
              fill: "currentColor",
              ...children.props,
            })}
      </button>
    );
  },
);
