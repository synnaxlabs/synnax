// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, UnknownRecord } from "@synnaxlabs/x";

import data from "@/hardware/ni/device/enrich/enriched.json";
import {
  Properties,
  PropertiesDigest,
  ZERO_PROPERTIES,
} from "@/hardware/ni/device/types";

type PickedEnrichedProperties = Pick<
  Properties,
  | "analogInput"
  | "analogOutput"
  | "digitalInputOutput"
  | "digitalInput"
  | "digitalOutput"
>;

export const enrich = (model: string, info: PropertiesDigest): Properties => {
  if (info.enriched === true) return info as Properties;
  const enriched = (data as unknown as UnknownRecord)[model] as {
    estimatedPinout: PickedEnrichedProperties;
  };
  if (enriched == null)
    return {
      ...deep.copy(ZERO_PROPERTIES),
      ...info,
      enriched: true,
    };
  return {
    ...info,
    ...enriched.estimatedPinout,
    enriched: true,
  } as Properties;
};
