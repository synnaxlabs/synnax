import { HTMLAttributes, createElement, forwardRef } from "react";

import clsx from "clsx";

import { TypographyLevel } from "./types";

export interface CoreTextProps {
  /* The level of text to display i.e. p, h1, h2 */
  level: TypographyLevel;
  /* The text to display */
  children?: string | number;
  /* The color of the text */
  color?: string;
}

export interface TextProps
  extends CoreTextProps,
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

export const Text = forwardRef(
  ({ level = "h1", color, className, style, ...props }: TextProps, ref) =>
    createElement(typographyLevelTags[level], {
      ref,
      style: { color, ...style },
      className: clsx(`pluto-text pluto-text--${level}`, className),
      ...props,
    })
);
Text.displayName = "Text";
