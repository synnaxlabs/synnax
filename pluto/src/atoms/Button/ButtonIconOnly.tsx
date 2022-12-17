import { cloneElement, ReactElement } from "react";

import clsx from "clsx";

import { BaseButtonProps } from "./Button";

/** The props for the {@link ButtonIconOnly} */
export interface ButtonIconOnlyProps extends BaseButtonProps {
  children: ReactElement;
}

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
export const ButtonIconOnly = ({
  children,
  className,
  variant = "text",
  size = "medium",
  ...props
}: ButtonIconOnlyProps): JSX.Element => {
  return (
    <button
      className={clsx(
        "pluto-btn pluto-btn-icon",
        "pluto-btn--" + size,
        "pluto-btn--" + variant,
        className
      )}
      {...props}
    >
      {cloneElement(children, { className: "pluto-btn-icon__icon" })}
    </button>
  );
};
