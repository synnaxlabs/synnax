// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device } from "@synnaxlabs/client";
import { strings } from "@synnaxlabs/x";

import { Device } from "@/platform/device";

/**
 * @returns the model of the device as an identifier, such as "t7", then the model with
 * the last four digits of its serial number, such as "t7_0002".
 */
export const suggestIdentifiers = ({ key, model }: device.Device): string[] => {
  const short = strings.trimPrefix(model, "LJM_dt").toLowerCase();
  return [short, `${short}_${key.slice(-4)}`].filter(
    (id) => Device.identifierZ.safeParse(id).success,
  );
};
