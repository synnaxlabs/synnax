// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button as CoreButton } from "@/core/Button/Button";
import { ButtonIcon } from "@/core/Button/ButtonIcon";
import { ButtonLink } from "@/core/Button/ButtonLink";
import { ButtonToggle, ButtonToggleIcon } from "@/core/Button/ButtonToggle";
export type { ButtonProps } from "@/core/Button/Button";
export type { ButtonIconProps } from "@/core/Button/ButtonIcon";
export type { ButtonLinkProps } from "@/core/Button/ButtonLink";

type CoreButtonType = typeof CoreButton;

interface ButtonType extends CoreButtonType {
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
  Icon: typeof ButtonIcon;
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
  Toggle: typeof ButtonToggle;
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
  ToggleIcon: typeof ButtonToggleIcon;
  /**
   * Button.Link renders a button that looks like a link and redirects to the given href
   * when clicked.
   *
   * @param props - Props for the component. Identical to the props for the Button component,
   * excluding 'variant', and  adding the following:
   * @param props.href - The URL to redirect to when the button is clicked.
   * @param props.target - The target of the link. Defaults to "_self".
   */
  Link: typeof ButtonLink;
}

/**
 * Button is a basic button component.
 *
 * @param props - Props for the component, which are passed down to the underlying button
 * element.
 * @param props.size - The size of button render.
 * @param props.variant - The variant to render for the button. Options are "filled"
 * (default), "outlined", and "text".
 * @param props.startIcon - An optional icon to render before the start of the button
 * text. This can be a single icon or an array of icons. The icons will be formatted
 * to match the color and size of the button.
 * @param props.endIcon - The same as {@link startIcon}, but renders after the button
 * text.
 */
export const Button = CoreButton as ButtonType;

Button.Icon = ButtonIcon;
Button.Toggle = ButtonToggle;
Button.ToggleIcon = ButtonToggleIcon;
Button.Link = ButtonLink;
