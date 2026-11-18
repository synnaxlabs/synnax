// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, type secret, type Synnax } from "@synnaxlabs/client";

import { MAKE, type Properties } from "@/feature/sift/device/types";
import { createTestDevice } from "@/platform/device/testutil";
import { uniqueName } from "@/testutil";

export interface CreateSiftDeviceOptions {
  rack?: rack.Key;
  properties?: Partial<Properties>;
}

/** Creates a stored secret that holds a Sift API key on the live Core. */
export const createAPIKey = async (client: Synnax): Promise<secret.Secret> =>
  await client.secrets.set({
    name: uniqueName("sift_api_key"),
    source: "stored",
    value: "key",
  });

/** Creates a Sift device on the live Core, with a new API key unless one is given. */
export const createSiftDevice = async (
  client: Synnax,
  { rack, properties }: CreateSiftDeviceOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    name: uniqueName("sift"),
    make: MAKE,
    model: "sift",
    rack,
    properties: {
      uri: "localhost:50051",
      apiKey: properties?.apiKey ?? (await createAPIKey(client)).key,
      ...properties,
    },
  });
