// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x";

/** DEFAULT_STROKE_WIDTH is the stroke width, in pixels, of a symbol that stores none. */
export const DEFAULT_STROKE_WIDTH = 2;

/** DEFAULT_BOX_BORDER_RADIUS is the corner radius, in pixels, of a box that stores none. */
export const DEFAULT_BOX_BORDER_RADIUS = 3;

/** DEFAULT_PRECISION is the number of decimal places a symbol shows when it stores none. */
export const DEFAULT_PRECISION = 2;

/** DEFAULT_BOUNDS is the range a gauge or scale indicator maps when it stores none. */
export const DEFAULT_BOUNDS: bounds.Bounds = { lower: 0, upper: 100 };
