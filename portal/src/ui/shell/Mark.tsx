// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, type ReactElement } from "react";

/* Even coverage of the OKLCH wheel. */
const HUES: readonly number[] = [15, 60, 105, 150, 195, 240, 285, 330];

const hueOf = (name: string): number => {
  let hash = 0;
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  return HUES[hash % HUES.length];
};

/** initials reads up to two letters from a name: "Acme Rockets" is "AR". */
const initials = (name: string): string =>
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w.charAt(0).toUpperCase())
    .join("") || "?";

export interface MarkProps {
  name: string;
  /** image replaces the initials when set. */
  image?: string;
}

/** Mark is a square with a frosted ring that stands for a user or a team. */
export const Mark = ({ name, image }: MarkProps): ReactElement => (
  <span
    className="portal-mark"
    style={{ "--portal-mark-hue": hueOf(name) } as CSSProperties}
  >
    <span className="portal-mark__fill">
      {image == null ? initials(name) : <img src={image} alt="" />}
    </span>
  </span>
);
