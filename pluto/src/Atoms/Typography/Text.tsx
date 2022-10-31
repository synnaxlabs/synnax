import { createElement, forwardRef, HTMLAttributes } from "react";
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
  ({ level = "h1", color, ...props }: TextProps, ref) => {
    props.style = { color, ...props.style };
    props.className = `pluto-text pluto-text--${level} ${props.className}`;
    return createElement(typographyLevelTags[level], { ref, ...props });
  }
);
Text.displayName = "Text";
