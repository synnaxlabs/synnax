// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, type Synnax } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import { Device } from "@/feature/http/device";
import { createTestDevice } from "@/platform/device/testutil";
import { uniqueName } from "@/testutil";

export interface CreateHTTPDeviceOptions {
  configured?: boolean;
  properties?: Partial<Device.Properties>;
  rack?: rack.Key;
}

/** Creates a configured HTTP server device on the live Core. */
export const createHTTPDevice = async (
  client: Synnax,
  { configured = true, properties, rack }: CreateHTTPDeviceOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    rack,
    name: uniqueName("http_server"),
    make: Device.MAKE,
    model: "HTTP server",
    configured,
    properties: { ...deep.copy(Device.ZERO_PROPERTIES), ...properties },
  });
