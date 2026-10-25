// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeTimeSpan,
  type DynamicWindow,
  type StaticWindow,
  type TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";

/** @returns a fixed window over the time range, with exact bounds. */
export const staticWindow = (tr: TimeRange): StaticWindow => ({
  variant: "static",
  timeRange: { start: String(tr.start.valueOf()), end: String(tr.end.valueOf()) },
});

/** @returns a window of the span that ends now. */
export const dynamicWindow = (span: CrudeTimeSpan): DynamicWindow => ({
  variant: "dynamic",
  timeSpan: Number(new TimeSpan(span).valueOf()),
});
