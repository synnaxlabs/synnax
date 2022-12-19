import { TextDateTime } from "./DateTime";
import { Text as CoreText } from "./Text";
import { TextEditable } from "./TextEditable";
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
  Editable: typeof TextEditable;
  DateTime: typeof TextDateTime;
}

export const Text = CoreText as TextType;

Text.WithIcon = TextWithIcon;
Text.Editable = TextEditable;
Text.DateTime = TextDateTime;
