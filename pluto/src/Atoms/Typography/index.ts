import { Text as CoreText } from "./Text";
import { TextWithIcon } from "./TextWithIcon";
export type { CoreTextProps, TextProps } from "./Text";
export type { Size, TypographyDefinition, TypographyLevel } from "./types";
export {
  ComponentSizeTypographyLevels,
  TypographyLevelComponentSizes,
  TypographyLevels,
} from "./types";

type CoreTextType = typeof CoreText;

interface TextType extends CoreTextType {
  WithIcon: typeof TextWithIcon;
}

export const Text = CoreText as TextType;

Text.WithIcon = TextWithIcon;
