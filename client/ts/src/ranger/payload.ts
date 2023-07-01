// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

export type RangeKey = string;
export type RangeName = string;
export type ChannelKeys = string[];
export type ChannelNames = string[];
export type RangeParams = RangeKey | RangeName | ChannelKeys | ChannelNames;

export const rangePayload = z.object({
  key: z.string(),
  name: z.string(),
  timeRange: TimeRange.z,
});
