import { ButtonHTMLAttributes, PropsWithChildren } from "react";
import "./Button.css";
import { classList } from "../../util/css";

interface ButtonProps
  extends PropsWithChildren<ButtonHTMLAttributes<HTMLButtonElement>> {
  variant?: "filled" | "outlined";
  size?: "small" | "medium";
}

export default function Button({
  children,
  size = "medium",
  variant = "filled",
  className,
  ...props
}: ButtonProps) {
  return (
    <button
      className={classList(
        "pluto-btn",
        "pluto-btn--" + variant,
        "pluto-btn--" + size,
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
}
