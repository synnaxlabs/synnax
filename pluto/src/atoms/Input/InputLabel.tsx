import { DetailedHTMLProps, HTMLAttributes } from "react";

import clsx from "clsx";

import "./InputLabel.css";

export interface InputLabelProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLLabelElement>, HTMLLabelElement> {
  label?: string;
}

export const InputLabel = (props: InputLabelProps): JSX.Element => {
  return <label className={clsx("pluto-input-label", props.className)} {...props} />;
};
