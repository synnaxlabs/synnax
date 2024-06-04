// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { text } from "@/text/core";
import { type ThemeSpec } from "@/theming/core/theme";
import { type ComponentSize, isComponentSize } from "@/util/component";

export const fontString = (
  theme: ThemeSpec,
  level: text.Level | ComponentSize,
  weight?: text.Weight,
): string => {
  const {
    typography,
    sizes: { base },
  } = theme;
  const size =
    typography[isComponentSize(level) ? text.ComponentSizeLevels[level] : level].size;
  const sizePx = base * size;
  const [family, serif] = typography.family.split(", ");
  if (weight != null) return ` ${weight} ${sizePx}px ${family}, ${serif}`;
  return ` ${sizePx}px ${typography.family}`;
};
