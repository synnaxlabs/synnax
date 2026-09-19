// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device as PDevice, Form } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";

import { type Device, SCHEMAS } from "@/feature/http/device/types";

export const { use, useResult } = PDevice.createRetrieve(SCHEMAS);

/** The device the form's config names, or undefined until it resolves or on failure. */
export const useFromConfig = (): Device | undefined => {
  const key = Form.useFieldValue<device.Key>("config.device", { optional: true });
  return useResult(primitive.isNonZero(key) ? { key } : null).data;
};
