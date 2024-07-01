// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { type CrudeTimeSpan, toArray } from "@synnaxlabs/x";

interface BaseRange {
  name: string;
  key: string;
  persisted: boolean;
}

export type StaticRange = BaseRange & {
  variant: "static";
  timeRange: { start: number; end: number };
};

export type DynamicRange = BaseRange & {
  variant: "dynamic";
  span: CrudeTimeSpan;
};

export type Range = StaticRange | DynamicRange;

export const fromClientRange = (ranges: ranger.Range | ranger.Range[]): Range[] =>
  toArray(ranges).map((range) => ({
    variant: "static",
    key: range.key,
    name: range.name,
    timeRange: {
      start: Number(range.timeRange.start.valueOf()),
      end: Number(range.timeRange.end.valueOf()),
    },
    persisted: true,
  }));
