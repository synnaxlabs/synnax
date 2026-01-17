// Copyright 2026 Synnax Labs, Inc.
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

interface PickedEnrichedProperties extends Pick<
  Properties,
  | "analogInput"
  | "analogOutput"
  | "counterInput"
  | "digitalInputOutput"
  | "digitalInput"
  | "digitalOutput"
> {}

export const enrich = (model: string, properties: Properties): Properties => {
  const enriched = (data as record.Unknown)[model] as {
    estimatedPinout: PickedEnrichedProperties;
  };
  return { ...deep.copy(ZERO_PROPERTIES), ...enriched?.estimatedPinout, ...properties };
};
