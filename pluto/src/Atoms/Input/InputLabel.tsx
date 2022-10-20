import clsx from "clsx";
import { DetailedHTMLProps, HTMLAttributes, LabelHTMLAttributes } from "react";
import "./InputLabel.css";

export interface InputLabelProps
  extends DetailedHTMLProps<
    HTMLAttributes<HTMLLabelElement>,
    HTMLLabelElement
  > {
  label?: string;
}

export default function InputLabel(props: InputLabelProps) {
  return (
    <label className={clsx("pluto-input-label", props.className)} {...props} />
  );
}
