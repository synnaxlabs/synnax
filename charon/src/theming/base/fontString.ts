// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component } from "@/component";
import { isSize } from "@/component/size";
import { text } from "@/text/base";
import { type ThemeSpec } from "@/theming/base/theme";

interface FontStringOptions {
  level: text.Level | Component.Size;
  weight?: text.Weight;
  code?: boolean;
}

type FontThreshold = {
  weight: number;
  style?: "light" | "regular" | "medium" | "bold";
  value: string;
};

const thresholds: FontThreshold[] = [
  { weight: 200, style: "light", value: "Inter 200, sans-serif" },
  { weight: 300, value: "Inter Threee, sans-serif" },
  { weight: 400, value: "Inter Four, sans-serif" },
  { weight: 500, value: "Inter Five, sans-serif" },
  { weight: 600, value: "Inter Six, sans-serif" },
  { weight: 700, value: "Inter Seven, sans-serif" },
  { weight: 800, value: "Inter Eight, sans-serif" },
  { weight: 900, style: "bold", value: "Inter Nine, sans-serif" },
];

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
  { level, weight, code }: FontStringOptions,
): string => {
  const {
    typography,
    sizes: { base },
  } = theme;
  let fmly = typography.family;
  if (code) fmly = typography.codeFamily;
  else if (weight != null) {
    const threshold = matchThreshold(weight);
    if (threshold != null) fmly = threshold.value;
    else fmly = "Inter Light, sans-serif";
  }
  const size =
    typography[isSize(level) ? text.COMPONENT_SIZE_LEVELS[level] : level].size;
  const sizePx = (base * size).toFixed(1);
  const [family, serif] = fmly.split(", ");
  if (weight != null) return ` ${weight} ${sizePx}px ${family}, ${serif}`;
  return ` ${sizePx}px ${fmly}`;
};
