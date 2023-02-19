// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/pluto";

import {
  MultiXAxisRecord,
  MultiYAxisRecord,
  Vis,
  XAxisKey,
  XAxisRecord,
} from "../../types";

import { Range } from "@/features/workspace";

export interface LineVis extends Vis {
  channels: XAxisRecord & MultiYAxisRecord;
  ranges: MultiXAxisRecord;
  zoom: XY;
  pan: XY;
}

export interface LineSVis extends Omit<LineVis, "ranges"> {
  ranges: Record<XAxisKey, Range[]>;
}
