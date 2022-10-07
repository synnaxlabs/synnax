import { createElement, HTMLAttributes } from "react";
import { FontLevel } from "../../Theme/theme";

export interface BaseTextProps {
  level: FontLevel;
  children?: string | number;
}

export interface TextProps
  extends BaseTextProps,
    Omit<HTMLAttributes<HTMLParagraphElement>, "children"> {}

const levelTag = {
  h1: "h1",
  h2: "h2",
  h3: "h3",
  h4: "h4",
  h5: "h5",
  p: "p",
  small: "h6",
};

export type Size = "small" | "medium" | "large";

export const sizeLevels: Record<Size, FontLevel> = {
  small: "small",
  medium: "p",
  large: "h4",
};

export default function Text({ level = "h1", ...props }: TextProps) {
  return createElement(levelTag[level], { ...props });
}
