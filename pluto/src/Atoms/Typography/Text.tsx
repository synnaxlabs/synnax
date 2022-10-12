import { createElement, HTMLAttributes } from "react";
import TextWithIcon from "./TextWithIcon";
import { TypographyLevel } from "./Types";

export interface BaseTextProps {
  /* The level of text to display i.e. p, h1, h2 */
  level: TypographyLevel;
  /* The text to display */
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

/* Displays text with a particular level */
function Text({ level = "h1", ...props }: TextProps) {
  return createElement(typographyLevelTags[level], props);
}

Text.WithIcon = TextWithIcon;

export default Text;
