import { DetailedHTMLProps, InputHTMLAttributes } from "react";

import { ComponentSize } from "@/util/component";

export type InputValue = boolean | string | number | readonly string[];

export interface InputControlProps<T extends InputValue = InputValue> {
  value: T;
  onChange: (value: T) => void;
}

type HTMLInputProps = Omit<
  DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>,
  "ref" | "size" | "onChange" | "value" | "children"
>;

export interface InputBaseProps<T extends InputValue = InputValue>
  extends HTMLInputProps,
    InputControlProps<T> {
  size?: ComponentSize;
}
