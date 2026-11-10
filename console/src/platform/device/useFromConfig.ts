// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type Device, type Flux } from "@synnaxlabs/pluto";
import { deep, primitive, type state } from "@synnaxlabs/x";

const path = deep.path<{ config: { device: device.Key } }>();

/**
 * Creates a hook that returns the device the form's config names, or undefined until it
 * resolves or on failure.
 * @param useResult - The vendor's typed device retrieve result hook.
 */
export const createUseFromConfig = <D extends state.State>(
  useResult: Flux.UseResult<Device.RetrieveQuery, D>,
) => {
  const useFromConfig = (): D | undefined => {
    const key = Form.useFieldValue(path.config.device, { optional: true });
    return useResult(primitive.isNonZero(key) ? { key } : null).data;
  };
  return useFromConfig;
};
