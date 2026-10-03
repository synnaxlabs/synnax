// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Pair } from "@/ranger/kv/types.gen";

export const SET_CHANNEL_NAME = "sy_range_kv_set";
export const DELETE_CHANNEL_NAME = "sy_range_kv_delete";

export const createPairKey = ({ range, key }: Omit<Pair, "value">) =>
  `${range}<--->${key}`;
