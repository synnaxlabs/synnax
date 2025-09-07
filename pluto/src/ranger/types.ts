// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TimeRange, TimeStamp } from "@synnaxlabs/x";

export const HAUL_TYPE = "range";

export const STAGES = ["to_do", "in_progress", "completed"] as const;

export type Stage = (typeof STAGES)[number];

export const getStage = (timeRange: TimeRange): Stage => {
  const now = TimeStamp.now();
  const tr = timeRange.makeValid();
  if (now.before(tr.start)) return "to_do";
  if (now.after(tr.end)) return "completed";
  return "in_progress";
};
