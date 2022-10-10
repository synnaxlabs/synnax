import { createElement, HTMLAttributes, PropsWithChildren } from "react";

export type textLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";

export interface TextProps extends PropsWithChildren<HTMLAttributes<any>> {
  level: textLevel;
  children: string | number;
}

const levelTag = {
  h1: "h1",
  h2: "h2",
  h3: "h3",
  h4: "h4",
  h5: "h5",
  p: "p",
  small: "h6",
};

const defaultLevel = "h1"

const Text = ({ level = defaultLevel, ...props }: TextProps) => {
  return createElement(levelTag[level] || defaultLevel, { ...props });
}


export default Text;
