import { ComponentSize } from "../../util/types";

export type Size = number | string;

export type TypographyLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";

export type TypographyDefinition = {
  size: Size;
  weight: Size;
  lineHeight: number;
  textTransform?: string;
};

export const ComponentSizeTypographyLevels: Record<
  ComponentSize,
  TypographyLevel
> = {
  small: "small",
  medium: "p",
  large: "h1",
};
