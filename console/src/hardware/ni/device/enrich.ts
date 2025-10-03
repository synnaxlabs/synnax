// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type record } from "@synnaxlabs/x";

import data from "@/hardware/ni/device/enriched.json";
import { type Properties, ZERO_PROPERTIES } from "@/hardware/ni/device/types";

interface PickedEnrichedProperties
  extends Pick<
    Properties,
    | "analogInput"
    | "analogOutput"
    | "counter"
    | "digitalInputOutput"
    | "digitalInput"
    | "digitalOutput"
  > {}

export const enrich = (model: string, properties: Properties): Properties => {
  const enriched = (data as record.Unknown)[model] as {
    estimatedPinout: PickedEnrichedProperties;
  };
  const merged = { ...deep.copy(ZERO_PROPERTIES), ...enriched?.estimatedPinout, ...properties };

  // Migration: If counter property doesn't exist or has 0 ports, default to 2
  // (most NI DAQ devices have at least 2 counters)
  if (merged.counter.portCount === 0 && merged.analogInput.portCount > 0) 
    merged.counter.portCount = 2;
  

  return merged;
};
