// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { text } from "@synnaxlabs/x";

import { Component } from "@/component";
import type { ThemeSpec } from "@/theme/theme";

/** The canvas face: Inter, Inter with tabular digits ("Inter Tabular"), or code. */
export type FontVariant = "prose" | "numeric" | "code";

interface FontStringOptions {
  level: text.Level | Component.Size;
  weight?: text.Weight;
  variant?: FontVariant;
}

type FontThreshold = {
  weight: number;
  style?: "light" | "regular" | "medium" | "bold";
  value: string;
};

const thresholds: FontThreshold[] = [
  { weight: 200, style: "light", value: "Inter Two, sans-serif" },
  { weight: 300, value: "Inter Three, sans-serif" },
  { weight: 400, value: "Inter Four, sans-serif" },
  { weight: 500, value: "Inter Five, sans-serif" },
  { weight: 600, value: "Inter Six, sans-serif" },
  { weight: 700, value: "Inter Seven, sans-serif" },
  { weight: 800, value: "Inter Eight, sans-serif" },
  { weight: 900, style: "bold", value: "Inter Nine, sans-serif" },
];

const NUMERIC_FAMILY = "Inter Tabular, sans-serif";

const matchThreshold = (weight: text.Weight): FontThreshold | undefined =>
  thresholds.find((t, i) => {
    const isLast = i === thresholds.length - 1;
    if (isLast) return true;
    if (typeof weight === "number")
      return weight >= t.weight && weight < thresholds[i + 1].weight;
    return t.style === weight;
  });

export const fontString = (
  theme: ThemeSpec,
  { level, weight, variant = "prose" }: FontStringOptions,
): string => {
  const {
    typography,
    sizes: { base },
  } = theme;
  let fmly = typography.family;
  if (variant === "code") fmly = typography.codeFamily;
  else if (variant === "numeric") fmly = NUMERIC_FAMILY;
  else if (weight != null) {
    const threshold = matchThreshold(weight);
    if (threshold != null) fmly = threshold.value;
    else fmly = "Inter Light, sans-serif";
  }
  const size =
    typography[Component.isSize(level) ? Component.SIZE_TEXT_LEVELS[level] : level]
      .size;
  const sizePx = (base * size).toFixed(1);
  const [family, serif] = fmly.split(", ");
  if (weight != null) return ` ${weight} ${sizePx}px ${family}, ${serif}`;
  return ` ${sizePx}px ${fmly}`;
};
