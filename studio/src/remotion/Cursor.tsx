// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

export interface CursorProps {
  left: number;
  top: number;
  scale: number;
  pressed: boolean;
  dsf: number;
}

/** Base cursor height in CSS px before scaling; drawn 1.5x natural size. */
const CURSOR_HEIGHT = 24 * 1.5;

/**
 * Cursor draws our own vector arrow (no OS cursor assets) hotspot-anchored at
 * its tip, sized in output pixels so it stays constant on screen through zooms.
 */
export const Cursor = ({
  left,
  top,
  scale,
  pressed,
  dsf,
}: CursorProps): ReactElement => {
  const h = CURSOR_HEIGHT * dsf * scale;
  return (
    <svg
      viewBox="0 0 20 24"
      style={{
        position: "absolute",
        left,
        top,
        height: h,
        transformOrigin: "0 0",
        filter: `drop-shadow(0 ${dsf}px ${2 * dsf}px rgba(0,0,0,${pressed ? 0.5 : 0.35}))`,
      }}
    >
      <path
        d="M 2 1 L 2 19 L 6.6 14.6 L 9.4 21.5 L 12.8 20 L 10 13.4 L 16.4 13 Z"
        fill="white"
        stroke="black"
        strokeWidth={1.4}
        strokeLinejoin="round"
      />
    </svg>
  );
};
