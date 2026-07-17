// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";

export interface ChangeIdentifierParams {
  key: device.Key;
  identifier: string;
}

export const { useUpdate: useChangeIdentifier } =
  Flux.createUpdate<ChangeIdentifierParams>({
    name: "device identifier",
    verbs: {
      present: "change identifier",
      past: "changed identifier",
      participle: "changing identifier",
    },
    update: async ({ client, data }) => {
      const { key, identifier } = data;
      const d = await Device.retrieveSingle({ client, query: { key } });
      await client.devices.create({
        ...d,
        properties: { ...d.properties, identifier },
      });
      return data;
    },
  });
