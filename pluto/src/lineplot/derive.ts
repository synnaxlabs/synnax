// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";

// RawDerivedLine is a stored line enriched with its decoded identity (axis,
// range, and channel keys parsed from Line.key). Its color may be unset.
export interface RawDerivedLine extends lineplot.Line, lineplot.LineKeyParts {}

// DerivedLine is a RawDerivedLine with its render color resolved to a concrete
// palette color, ready for the chart and toolbar to consume directly.
export interface DerivedLine extends Omit<RawDerivedLine, "color"> {
  color: color.Color;
}

// resolveLineColor returns the concrete color a line should render with: its
// stored color when set, otherwise a palette color chosen by its position. The
// chart and the toolbar both route through this so the displayed colors agree.
export const resolveLineColor = (
  stored: color.Color | undefined,
  index: number,
  palette: color.Crude[],
): color.Color =>
  stored ?? color.construct(palette[index % Math.max(palette.length, 1)] ?? color.ZERO);
