import { Button as CoreButton } from "./Button";
import { ButtonIconOnly } from "./ButtonIconOnly";
export type { ButtonProps } from "./Button";
export type { ButtonIconOnlyProps } from "./ButtonIconOnly";

type CoreButtonType = typeof CoreButton;

interface ButtonType extends CoreButtonType {
  /**
   * Button.IconOnly is a button that only renders an icon without any text.
   *
   * @param props - Props for the component, which are passed down to the underlying
   * element.
   * @param props.size - The size of button to render.
   * @param props.variant - The variant of button to render. Options are "filled" (default),
   * "outlined", and "text".
   * @param props.children - A ReactElement representing the icon to render.
   */
  IconOnly: typeof ButtonIconOnly;
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

Button.IconOnly = ButtonIconOnly;
