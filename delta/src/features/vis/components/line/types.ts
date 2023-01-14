// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/pluto";

import { Vis } from "../types";

import { Range } from "@/features/workspace";

export interface LineVis extends Vis {
  channels: {
    y1: readonly string[];
    y2: readonly string[];
    y3: readonly string[];
    y4: readonly string[];
    x1: string;
  };
  ranges: {
    x1: readonly string[];
  };
  zoom: XY;
  pan: XY;
}

export interface LineSVis extends Omit<LineVis, "ranges"> {
  ranges: {
    x1: Range[];
  };
}

export class EnhancedLinePlotVS {
  vs: LineSVis;

  constructor(vs: LineSVis) {
    this.vs = vs;
  }

  get ranges(): Range[] {
    return this.vs.ranges.x1;
  }

  get keys(): string[] {
    const { channels } = this.vs;
    return Object.values(channels)
      .flat()
      .filter((key) => key.length > 0);
  }
}

export interface Line {
  y: string;
  x: string;
  color: string;
}
