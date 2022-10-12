import { ComponentSize } from "../../util/types";

export type Size = number | string;

/* Level of typography i.e paragraph and heading */
export type TypographyLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
export const TypographyLevels = ["h1", "h2", "h3", "h4", "h5", "p", "small"];

/* Defines a particular typography style */
export type TypographyDefinition = {
  size: Size;
  weight: Size;
  lineHeight: number;
  textTransform?: string;
};

/* Standardizes the typography levels for components of different sizes */
export const ComponentSizeTypographyLevels: Record<
  ComponentSize,
  TypographyLevel
> = {
  small: "small",
  medium: "p",
  large: "h1",
};

export const TypographyLevelComponentSizes: Record<
  TypographyLevel,
  ComponentSize
> = {
  h1: "large",
  h2: "large",
  h3: "medium",
  h4: "medium",
  h5: "small",
  p: "small",
  small: "small",
};
