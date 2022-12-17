import { Button as CoreButton } from "./Button";
import { ButtonIconOnly } from "./ButtonIconOnly";
export type { ButtonProps } from "./Button";
export type { ButtonIconOnlyProps } from "./ButtonIconOnly";

type CoreButtonType = typeof CoreButton;

interface ButtonType extends CoreButtonType {
  IconOnly: typeof ButtonIconOnly;
}

export const Button = CoreButton as ButtonType;

Button.IconOnly = ButtonIconOnly;
