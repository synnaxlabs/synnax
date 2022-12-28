import { Text as CoreText } from "./Text";
import { TextDateTime } from "./TextDateTime";
import { TextEditable } from "./TextEditable";
import { TextWithIcon } from "./TextWithIcon";
import {
  ComponentSizeTypographyLevels,
  TypographyLevelComponentSizes,
  TypographyLevels,
} from "./types";
export type { CoreTextProps, TextProps } from "./Text";
export type { Size, TypographyDefinition, TypographyLevel } from "./types";
export type { TextWithIconProps } from "./TextWithIcon";

type CoreTextType = typeof CoreText;

interface TextType extends CoreTextType {
  WithIcon: typeof TextWithIcon;
  Editable: typeof TextEditable;
  DateTime: typeof TextDateTime;
}

export const Text = CoreText as TextType;

Text.WithIcon = TextWithIcon;
Text.Editable = TextEditable;
Text.DateTime = TextDateTime;

export const Typography = {
  ComponentSizeLevels: ComponentSizeTypographyLevels,
  LevelComponentSizes: TypographyLevelComponentSizes,
  Levels: TypographyLevels,
  Text,
};
