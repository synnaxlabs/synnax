// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

/// symbolColorVar returns the value for the --pluto-symbol-color custom property: the
/// rgba channels of a symbol's color, or undefined for an unset or ZERO color so the
/// display var falls back to the theme default.
export const symbolColorVar = (c?: color.Crude): string | undefined =>
  c != null && !color.isZero(c)
    ? `${color.rgbString(c)}, ${color.aValue(c)}`
    : undefined;
