// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { tokens } from "@synnaxlabs/arc";
import { createCssVariablesTheme } from "shiki";

const arcRules = Object.values(tokens).flatMap(({ scopes, light, dark }) =>
  scopes.map((scope) => ({
    scope,
    settings: { foreground: `light-dark(${light}, ${dark})` },
  })),
);

/**
 * Astro's css-variables theme, extended with the Arc colors the Console uses. Every Arc
 * scope ends in `.arc`, so the added rules outrank the generic ones and reach no other
 * language. `light-dark()` needs `color-scheme` on the root element.
 */
export const theme = createCssVariablesTheme({ variablePrefix: "--astro-code-" });
theme.tokenColors = [...(theme.tokenColors ?? []), ...arcRules];
