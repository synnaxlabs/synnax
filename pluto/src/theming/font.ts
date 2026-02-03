// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Text } from "@/text";
import { fontString } from "@/theming/base/fontString";
import { useContext } from "@/theming/Provider";

export interface UseTypographyReturn extends Text.Spec {
  toString: () => string;
  baseSize: number;
  lineHeightPx: number;
  sizePx: number;
}

export const useTypography = (level: Text.Level): UseTypographyReturn => {
  const { theme } = useContext();
  const t = theme.typography[level];
  return {
    ...t,
    toString: () => fontString(theme, { level }),
    baseSize: theme.sizes.base,
    lineHeightPx: t.lineHeight * theme.sizes.base,
    sizePx: t.size * theme.sizes.base,
  };
};
