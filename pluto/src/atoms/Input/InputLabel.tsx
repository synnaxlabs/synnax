import clsx from "clsx";
import { DetailedHTMLProps, HTMLAttributes } from "react";

export interface InputLabelProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLLabelElement>, HTMLLabelElement> {
  label?: string;
}

export const InputLabel = (props: InputLabelProps) => {
  return <label className={clsx("pluto-input-label", props.className)} {...props} />;
};
