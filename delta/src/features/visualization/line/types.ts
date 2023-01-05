// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { LinePlotMetadata } from "@synnaxlabs/pluto";

import { Visualization } from "../types";

export interface LinePlotVisualization extends Visualization, LinePlotMetadata {
  channels: {
    y1: readonly string[];
    y2: readonly string[];
    y3: readonly string[];
    y4: readonly string[];
    x1: string;
    x2: string;
  };
  ranges: {
    x1: readonly string[];
    x2: readonly string[];
  };
}

export interface SugaredLinePlotVisualization
  extends Omit<LinePlotVisualization, "ranges"> {
  ranges: {
    x1: Range[];
    x2: Range[];
  };
}
