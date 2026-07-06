// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import { MAKE, type Properties, ZERO_PROPERTIES } from "@/feature/http/device/types";
import { createTestDevice } from "@/platform/device/testutil";
import { uniqueName } from "@/testutil";

export interface CreateHTTPDeviceOptions {
  configured?: boolean;
  properties?: Partial<Properties>;
}

/** Creates a rack and a configured HTTP server device on the live cluster. */
export const createHTTPDevice = async (
  client: Synnax,
  { configured = true, properties }: CreateHTTPDeviceOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    name: uniqueName("http_server"),
    make: MAKE,
    model: "HTTP server",
    configured,
    properties: { ...deep.copy(ZERO_PROPERTIES), ...properties },
  });
