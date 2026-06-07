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

// DerivedLine is a stored line enriched with its decoded identity (axis, range,
// and channel keys parsed from Line.key) for convenient consumption by the
// chart and toolbar.
export interface DerivedLine extends lineplot.Line, lineplot.LineKeyParts {}

// resolveLineColor returns the concrete color a line should render with: its
// stored color when set, otherwise a palette color chosen by its position. The
// chart and the toolbar both route through this so the displayed colors agree.
export const resolveLineColor = (
  stored: color.Color | undefined,
  index: number,
  palette: color.Crude[],
): color.Color =>
  stored ?? color.construct(palette[index % Math.max(palette.length, 1)] ?? color.ZERO);
