// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/project/Avatar.css";

import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/platform/css";

type Variant = "dusty" | "tint" | "solid" | "vivid";

const VARIANT: Variant = "dusty";

/* Hue strategies, all in OKLCH degrees:
   - full-wheel: even coverage, maximum distinguishability, includes muddy slots.
   - curated: full range but hand-picked, skips the 80-130 yellow-olive dead zone.
   - cool: clustered on the teal-violet half, harmonized with the chrome's 258 tint.
   - analogous: tight band around the brand hue; color as texture, not identity. */
const HUE_STRATEGIES = {
  "full-wheel": [15, 60, 105, 150, 195, 240, 285, 330],
  curated: [25, 55, 145, 195, 230, 262, 295, 330],
  cool: [160, 185, 210, 235, 258, 285, 310, 335],
  analogous: [218, 233, 248, 263, 278, 293],
} as const;

const HUES: readonly number[] = HUE_STRATEGIES["full-wheel"];

/* Hashes the full name, not the initial, so same-letter projects get different
   hues. */
const hueOf = (name: string): number => {
  let hash = 0;
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  return HUES[hash % HUES.length];
};

/* Multi-word names take each word's initial ("Hot Fire" -> "HF"); single words
   take their first two characters ("Primary" -> "PR"). */
const initialsOf = (name: string): string => {
  const words = name
    .trim()
    .split(/[\s\-_]+/)
    .filter(Boolean);
  if (words.length === 0) return "?";
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0].charAt(0) + words[1].charAt(0)).toUpperCase();
};

export interface AvatarProps {
  name: string;
}

export const Avatar = ({ name }: AvatarProps): ReactElement => {
  const hue = hueOf(name);
  const style = useMemo(
    () => ({ "--console-avatar-hue": hue }) as CSSProperties,
    [hue],
  );
  return (
    <div
      className={CSS(CSS.B("project-avatar"), CSS.BM("project-avatar", VARIANT))}
      style={style}
    >
      {initialsOf(name)}
    </div>
  );
};
