// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ComponentSizeTypographyLevels,
  TypographyLevel,
} from "@/core/std/Typography/types";
import { ThemeSpec } from "@/core/theming/theme";
import { ComponentSize, isComponentSize } from "@/util/component";

export const fontString = (
  theme: ThemeSpec,
  level: TypographyLevel | ComponentSize
): string => {
  const {
    typography,
    sizes: { base },
  } = theme;
  const size =
    typography[isComponentSize(level) ? ComponentSizeTypographyLevels[level] : level]
      .size;
  return ` ${base * size}px ${typography.family}`;
};
