import { createElement, HTMLAttributes } from "react";
import { TypographyLevel } from "./Types";

export interface BaseTextProps {
  level: TypographyLevel;
  children?: string | number;
}

export interface TextProps
  extends BaseTextProps,
    Omit<HTMLAttributes<HTMLParagraphElement>, "children"> {}

const typographyLevelTags = {
  h1: "h1",
  h2: "h2",
  h3: "h3",
  h4: "h4",
  h5: "h5",
  p: "p",
  small: "h6",
};

export default function Text({ level = "h1", ...props }: TextProps) {
  return createElement(typographyLevelTags[level], props);
}
