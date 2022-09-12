import { InputHTMLAttributes } from "react";
interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "size"> {
    size?: "small" | "medium";
    name?: string;
    label?: string;
}
declare const Input: ({ size, name, label, placeholder, value, ...props }: InputProps) => JSX.Element;
export default Input;
