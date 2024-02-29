// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import data from "@/hardware/configure/ni/enriched.json";
import {
  type EnrichedProperties,
  type PropertiesDigest,
} from "@/hardware/device/new/types";

export const enrich = (info: PropertiesDigest): EnrichedProperties => {
  const enriched = data[info.model];
  return {
    ...info,
    ...enriched.estimatedPinout,
  };
};
