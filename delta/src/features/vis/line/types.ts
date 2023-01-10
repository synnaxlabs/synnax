// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/client";
import { RGBATuple } from "@synnaxlabs/pluto";

import { Visualization } from "../types";

import { Range } from "@/features/workspace";

export interface LinePlotV extends Visualization {
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
}

export interface LinePlotVS extends Omit<LinePlotV, "ranges"> {
  ranges: {
    x1: Range[];
  };
}

export class EnhancedLinePlotVS {
  vs: LinePlotVS;

  constructor(vs: LinePlotVS) {
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

export interface Axis {
  key: YAxisKey;
  label: string;
  lines: Line[];
}
