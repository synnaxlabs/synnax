// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type location } from "@synnaxlabs/x";

export interface OrientableProps {
  orientation?: location.Outer;
}

export interface SVGBasedProps extends OrientableProps {
  strokeColor?: color.Crude;
  scale?: number;
}

export const ZERO_PROPS = {
  orientation: "left",
  scale: 1,
} as const satisfies SVGBasedProps;

/**
 * @returns the scale a symbol renders at. A stored scale that is absent, not finite,
 * or not positive would collapse the symbol to nothing, so it renders at 1 instead.
 */
export const resolveScale = (scale?: number): number =>
  scale != null && Number.isFinite(scale) && scale > 0 ? scale : 1;
