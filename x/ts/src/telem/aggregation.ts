// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Aggregation } from "@/telem/types.gen";

/** The number of points each group of samples reduces to under the aggregation. */
export const POINTS_PER_GROUP: Record<Aggregation, number> = {
  min_max: 2,
  average: 1,
  decimate: 1,
};
