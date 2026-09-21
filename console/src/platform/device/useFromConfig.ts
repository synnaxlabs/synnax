// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Device, type Flux, Form } from "@synnaxlabs/pluto";
import { primitive, type state } from "@synnaxlabs/x";

/**
 * Creates a hook that returns the device the form's config names, or undefined until
 * it resolves or on failure.
 * @param useResult - The vendor's typed device retrieve result hook.
 */
export const createUseFromConfig =
  <D extends state.State>(useResult: Flux.UseResult<Device.RetrieveQuery, D>) =>
  (): D | undefined => {
    const key = Form.useFieldValue<string>("config.device", { optional: true });
    return useResult(primitive.isNonZero(key) ? { key } : null).data;
  };
