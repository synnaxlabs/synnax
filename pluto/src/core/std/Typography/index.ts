// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Symbols } from "@/core/std/Typography/symbols";
import { Text as CoreText } from "@/core/std/Typography/Text";
import { TextDateTime } from "@/core/std/Typography/TextDateTime";
import { textDimensions } from "@/core/std/Typography/textDimensions";
import { TextMaybeEditable, TextEditable } from "@/core/std/Typography/TextEditable";
import { TextKeyboard } from "@/core/std/Typography/TextKeyboard";
import { TextLink } from "@/core/std/Typography/TextLink";
import { TextWithIcon } from "@/core/std/Typography/TextWithIcon";
import {
  ComponentSizeTypographyLevels,
  typographyLevel,
  TypographyLevelComponentSizes,
  TypographyLevels,
  typographySpec,
} from "@/core/std/Typography/types";

import "@/core/std/Typography/Typography.css";

export type { CoreTextProps, TextProps } from "@/core/std/Typography/Text";
export type { TypographySpec, TypographyLevel } from "@/core/std/Typography/types";
export type { TextWithIconProps } from "@/core/std/Typography/TextWithIcon";
export type { TextLinkProps } from "@/core/std/Typography/TextLink";
export type {
  TextEditableProps,
  TextMaybeEditableProps,
} from "@/core/std/Typography/TextEditable";

type CoreTextType = typeof CoreText;

interface TextType extends CoreTextType {
  WithIcon: typeof TextWithIcon;
  Editable: typeof TextEditable;
  MaybeEditable: typeof TextMaybeEditable;
  DateTime: typeof TextDateTime;
  Link: typeof TextLink;
  Keyboard: typeof TextKeyboard;
  dimensions: typeof textDimensions;
}

export const Text = CoreText as TextType;

Text.WithIcon = TextWithIcon;
Text.Editable = TextEditable;
Text.MaybeEditable = TextMaybeEditable;
Text.DateTime = TextDateTime;
Text.Link = TextLink;
Text.dimensions = textDimensions;
Text.Keyboard = TextKeyboard;

export interface TypographyType {
  ComponentSizeLevels: typeof ComponentSizeTypographyLevels;
  LevelComponentSizes: typeof TypographyLevelComponentSizes;
  Levels: typeof TypographyLevels;
  Text: TextType;
  spec: typeof typographySpec;
  levelZ: typeof typographyLevel;
  Symbols: typeof Symbols;
}

/** Holds typography related components and constants. */
export const Typography: TypographyType = {
  /** A map of component sizes to typography levels that are similar in size. */
  ComponentSizeLevels: ComponentSizeTypographyLevels,
  /** A map of typography levels to component sizes that are similar in size. */
  LevelComponentSizes: TypographyLevelComponentSizes,
  /** A list of all typography levels. */
  Levels: TypographyLevels,
  /** Renders text of a given typography level. */
  Text,
  /**  */
  spec: typographySpec,
  levelZ: typographyLevel,
  Symbols,
};
