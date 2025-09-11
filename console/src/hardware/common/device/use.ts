// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Form } from "@synnaxlabs/pluto";
import { primitive, type record } from "@synnaxlabs/x";
import { useEffect, useMemo } from "react";

/**
 * A hook that retrieves and subscribes to updates for a device. Must be used within a
 * Form context that has a schema matching the following structure:
 *
 * ```typescript
 * {
 *   config: {
 *     device: z.string() // The device key
 *   }
 * }
 * ```
 *
 * @returns The device if found, undefined otherwise.
 * @template P - The type of the device properties.
 * @template MK - The device make type.
 * @template MO - The device model type.
 */
export const use = <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>(): device.Device<Properties, Make, Model> | null => {
  const devKey = Form.useFieldValue<string>("config.device");
  const { useRetrieveStateful } = useMemo(
    () => Device.createRetrieve<Properties, Make, Model>(),
    [],
  );
  const { retrieve, data } = useRetrieveStateful();
  useEffect(() => {
    if (primitive.isZero(devKey)) return;
    retrieve({ key: devKey });
  }, [devKey, retrieve]);
  return data ?? null;
};
