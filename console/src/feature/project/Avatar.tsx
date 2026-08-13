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

/* Even coverage of the OKLCH wheel. */
const HUES: readonly number[] = [15, 60, 105, 150, 195, 240, 285, 330];

/* Hashes the full name, not the initials, so same-letter projects differ. */
const hueOf = (name: string): number => {
  let hash = 0;
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  return HUES[hash % HUES.length];
};

/* "Hot Fire" -> "HF"; "Primary" -> "PR"; "Test Stand 2" -> "T2", so numbered
   siblings do not collapse onto the same two letters. */
const initialsOf = (name: string): string => {
  const trimmed = name.trim();
  const words = trimmed.split(/[\s\-_]+/).filter(Boolean);
  if (words.length === 0) return "?";
  const first = words[0].charAt(0);
  const digit = trimmed.match(/(\d)$/)?.[1];
  if (digit != null && digit !== first) return (first + digit).toUpperCase();
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (first + words[1].charAt(0)).toUpperCase();
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
    <div className={CSS.B("project-avatar")} style={style}>
      {initialsOf(name)}
    </div>
  );
};
