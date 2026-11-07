// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type sift } from "@synnaxlabs/client";
import { bounds, TimeRange } from "@synnaxlabs/x";

/**
 * @returns the fraction of the upload's time range, from 0 to 1, before which Sift has
 * confirmed every sample.
 */
export const progress = ({ timeRange, positions }: sift.UploadDetails): number => {
  if (positions.length === 0) return 0;
  const tr = new TimeRange(timeRange);
  const reached = positions.reduce(
    (min, p) => (p.time.before(min) ? p.time : min),
    positions[0].time,
  );
  const done = new TimeRange(tr.start, reached).span.valueOf();
  return bounds.clamp({ lower: 0, upper: 1 }, Number(done) / Number(tr.span.valueOf()));
};
