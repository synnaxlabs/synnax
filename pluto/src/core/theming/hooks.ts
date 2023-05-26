// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useThemeContext } from "./ThemeContext";

import { TypographyLevel } from "@/core";

export const useFont = (level: TypographyLevel): string => {
  const {
    theme: {
      typography,
      sizes: { base },
    },
  } = useThemeContext();
  const { size } = typography[level];
  return `${(size as number) * base}px ${typography.family}`;
};
