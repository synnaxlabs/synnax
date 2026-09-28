// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/avatar/Avatar.css";

import { type ReactElement, useMemo } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";

/* Even coverage of the OKLCH wheel. */
const HUES: readonly number[] = [15, 60, 105, 150, 195, 240, 285, 330];

/* Hashes the full name, not the initials, so same-letter names differ. */
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
  /** name picks the color and the initials. */
  name: string;
  /** image replaces the initials when set. */
  image?: string;
  size?: Component.Size;
  className?: string;
}

/**
 * Avatar is a square that stands for a named thing, such as a user or a project. It
 * shows the initials of the name, or an image, inside a frosted ring.
 */
export const Avatar = ({
  name,
  image,
  size = "medium",
  className,
}: AvatarProps): ReactElement => {
  const hue = hueOf(name);
  const style = useMemo<CSS.VarProperties>(
    () => ({ "--pluto-avatar-hue": hue }),
    [hue],
  );
  return (
    <span
      className={CSS.cls(CSS.B("avatar"), CSS.M("height", size), className)}
      style={style}
    >
      <span className={CSS.BE("avatar", "fill")}>
        {image == null ? initialsOf(name) : <img src={image} alt="" />}
      </span>
    </span>
  );
};
