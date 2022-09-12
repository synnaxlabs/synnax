import { ButtonHTMLAttributes, PropsWithChildren } from "react";
interface ButtonProps extends PropsWithChildren<ButtonHTMLAttributes<HTMLButtonElement>> {
    variant?: "filled" | "outlined";
    size?: "small" | "medium";
}
export default function Button({ children, size, variant, className, ...props }: ButtonProps): JSX.Element;
export {};
