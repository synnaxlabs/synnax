// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

/** Ripple lifetime in output frames (~600 ms at 60fps). */
export const RIPPLE_TICKS = 36;

/** Maximum ripple radius in CSS px at 1x zoom. */
const MAX_RADIUS = 48;

export interface RippleProps {
  /** Normalized lifetime progress in [0, 1). */
  progress: number;
  left: number;
  top: number;
  amount: number;
  dsf: number;
}

/**
 * Ripple is the click highlight: a world-space circle expanding with an
 * ease-out cubic and fading from 30% opacity to zero.
 */
export const Ripple = ({
  progress,
  left,
  top,
  amount,
  dsf,
}: RippleProps): ReactElement => {
  const eased = 1 - (1 - progress) ** 3;
  const radius = MAX_RADIUS * eased * amount * dsf;
  const opacity = 0.3 * (1 - progress);
  return (
    <div
      style={{
        position: "absolute",
        left: left - radius,
        top: top - radius,
        width: radius * 2,
        height: radius * 2,
        borderRadius: "50%",
        background: `rgba(255, 255, 255, ${opacity})`,
        border: `${1.5 * dsf}px solid rgba(255, 255, 255, ${opacity * 1.5})`,
      }}
    />
  );
};
