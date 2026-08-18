// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type CursorKind } from "@/timeline";

export interface CursorProps {
  left: number;
  top: number;
  scale: number;
  pressed: boolean;
  dsf: number;
  kind: CursorKind;
  /** Idle fade opacity in [0, 1]. */
  opacity?: number;
}

/** Base cursor height in CSS px before scaling; drawn 1.5x natural size. */
const CURSOR_HEIGHT = 24 * 1.5;

interface Sprite {
  viewBox: string;
  /** Hotspot within the viewBox, subtracted so (left, top) lands on it. */
  hotspot: { x: number; y: number };
  path: string;
}

/**
 * Own vector artwork (no OS cursor assets): arrow for neutral hover, pointing
 * hand over clickables, I-beam over text, matching what a real recording of the
 * OS cursor would show.
 */
const SPRITES: Record<CursorKind, Sprite> = {
  default: {
    viewBox: "0 0 20 24",
    hotspot: { x: 2, y: 1 },
    path: "M 2 1 L 2 19 L 6.6 14.6 L 9.4 21.5 L 12.8 20 L 10 13.4 L 16.4 13 Z",
  },
  pointer: {
    viewBox: "0 0 24 28",
    hotspot: { x: 9.2, y: 1 },
    path: `M 7.5 4.4 C 7.5 2.5 8.3 1.2 9.6 1.2 C 10.9 1.2 11.7 2.5 11.7 4.4
      L 11.7 11.2 L 12.6 11.3 C 12.7 9.9 15.5 9.9 15.7 11.6 L 16.5 11.7
      C 16.7 10.4 19.2 10.5 19.4 12.2 L 20 12.3 C 20.3 11.2 22.3 11.3 22.4 13.2
      L 22.4 17.2 C 22.4 20.2 21.5 22 20.2 23.6 L 20.2 26.4 L 10.2 26.4
      L 10.2 24 C 8 21.8 5.3 19 4 16.6 C 3.2 15.1 4.5 13.7 6 14.2
      C 6.6 14.4 7.1 14.9 7.5 15.4 Z`,
  },
  text: {
    viewBox: "0 0 14 24",
    hotspot: { x: 7, y: 12 },
    path: `M 3 1.2 C 4.6 1.2 5.9 1.7 7 2.9 C 8.1 1.7 9.4 1.2 11 1.2 L 11 3.4
      C 9.6 3.4 8.1 3.7 8.1 5.9 L 8.1 18.1 C 8.1 20.3 9.6 20.6 11 20.6
      L 11 22.8 C 9.4 22.8 8.1 22.3 7 21.1 C 5.9 22.3 4.6 22.8 3 22.8
      L 3 20.6 C 4.4 20.6 5.9 20.3 5.9 18.1 L 5.9 5.9 C 5.9 3.7 4.4 3.4 3 3.4 Z`,
  },
};

/**
 * Cursor draws the synthetic cursor sprite hotspot-anchored at (left, top),
 * sized in output pixels so it stays constant on screen through zooms.
 */
export const Cursor = ({
  left,
  top,
  scale,
  pressed,
  dsf,
  kind,
  opacity = 1,
}: CursorProps): ReactElement => {
  const sprite = SPRITES[kind];
  const [, , vw, vh] = sprite.viewBox.split(" ").map(Number);
  const h = CURSOR_HEIGHT * dsf * scale;
  const px = h / vh;
  return (
    <svg
      viewBox={sprite.viewBox}
      style={{
        position: "absolute",
        left: left - sprite.hotspot.x * px,
        top: top - sprite.hotspot.y * px,
        width: vw * px,
        height: h,
        opacity,
        transformOrigin: "0 0",
        filter: `drop-shadow(0 ${dsf}px ${2 * dsf}px rgba(0,0,0,${pressed ? 0.5 : 0.35}))`,
      }}
    >
      <path
        d={sprite.path}
        fill="white"
        stroke="black"
        strokeWidth={1.4}
        strokeLinejoin="round"
      />
    </svg>
  );
};
