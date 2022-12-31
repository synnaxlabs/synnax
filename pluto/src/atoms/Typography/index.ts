// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
