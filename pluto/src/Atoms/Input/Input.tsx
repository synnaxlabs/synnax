import { InputHTMLAttributes, PropsWithChildren } from "react";
import { classList } from "../../util/css";
import "./Input.css";
import Space from "../Space/Space";
import { uuidShort } from "../../util/uuid";

interface InputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, "size"> {
  size?: "small" | "medium";
  name?: string;
  label?: string;
}

const Input = ({
  size = "medium",
  name = uuidShort(),
  label,
  placeholder,
  value,
  ...props
}: InputProps) => {
  return (
    <Space className="pluto-input__container" size={1}>
      {label && (
        <label className="pluto-input__label" htmlFor={name}>
          {label}
        </label>
      )}
      <input
        id={name}
        placeholder={placeholder}
        className={classList(
          "pluto-input__input",
          "pluto-input__input--" + size,
          props.className
        )}
        {...props}
      />
    </Space>
  );
}

export default Input;