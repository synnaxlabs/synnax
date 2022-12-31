// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { LinePlotMetadata } from "@synnaxlabs/pluto";

import { Range } from "@/features/workspace";

export interface Visualization {
  variant: string;
  layoutKey: string;
}

export interface LinePlotVisualization extends Visualization, LinePlotMetadata {
  channels: string[];
  ranges: string[];
}

export interface SugaredLinePlotVisualization
  extends Omit<LinePlotVisualization, "ranges"> {
  ranges: Range[];
}
