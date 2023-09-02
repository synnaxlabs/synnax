// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FunctionComponent } from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { Icon } from "@/button/Icon";
import { CSS } from "@/css";
import { type Input } from "@/input";

export interface ToggleExtensionProps extends Input.Control<boolean> {
  checkedVariant?: ButtonProps["variant"];
  uncheckedVariant?: ButtonProps["variant"];
}

const toggleFactory =
  <E extends Pick<ButtonProps, "className" | "variant">>(
    Base: FunctionComponent<E>,
  ): FunctionComponent<Omit<E, "value" | "onChange"> & ToggleExtensionProps> =>
  // eslint-disable-next-line react/display-name
  ({ value, checkedVariant = "filled", uncheckedVariant = "outlined", ...props }) => (
    // @ts-expect-error
    <Base
      {...props}
      checked={value}
      onClick={() => props.onChange(!value)}
      className={CSS(CSS.B("btn-toggle"), props.className)}
      variant={value ? checkedVariant : uncheckedVariant}
    />
  );

/**
 * Button.Toggle renders a button that can be toggled on and off. It implements the
 * InputControlProps interface, so it can be used as an input control in a form.
 *
 * @param props - Props for the component. Identical to the props for the Button
 * component, excluding 'variant', and  adding the following:
 * @param props.value - The boolean value of the button. If true, the button will be
 * rendered as "filled". If false, it will be rendered as "outlined".
 * @param props.onChange - A callback function that will be called when the button is
 * toggled. The callback will be passed the new value of the button.
 */
export const Toggle = toggleFactory(Button);
export type ToggleProps = Omit<Parameters<typeof Toggle>[0], "value" | "onChange"> &
  ToggleExtensionProps;
Toggle.displayName = "ButtonToggle";

/**
 * Button.IconToggle renders a button that can be toggled on and off, and only
 * renders an icon. It implements the InputControlProps interface, so it can be used
 *  as an input control in a form.
 *
 * @param props - Props for the component. Identical to the props for the Button.Icon
 * component, excluding 'variant', and  adding the following:
 * @param props.value - The boolean value of the button. If true, the button will be
 * rendered as "filled". If false, it will be rendered as "outlined".
 * @param props.onChange - A callback function that will be called when the button is
 * toggled. The callback will be passed the new value of the button.
 */

export const ToggleIcon = toggleFactory(Icon);
export type ToggleIconProps = Omit<
  Parameters<typeof ToggleIcon>[0],
  "value" | "onChange"
> &
  ToggleExtensionProps;
ToggleIcon.displayName = "ButtonToggleIcon";
