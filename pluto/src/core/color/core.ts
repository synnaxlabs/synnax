// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const hexRegex = /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i;
export const sixCharHexRegex = /^#?([0-9a-f]{6})$/i;
export const hex = z.string().regex(hexRegex);
export const sixCharHex = z.string().regex(sixCharHexRegex);

/** A completely transparent zero value color. */
export const ZERO_COLOR: RGBATuple = [0, 0, 0, 0];

/**
 * Represents a color in RGBA format. RGBA tuples can have any value range (0-255, 0-1, etc.).
 */
export type RGBATuple = [number, number, number, number];
