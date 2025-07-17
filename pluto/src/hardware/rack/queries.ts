// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";

export interface ListParams extends Flux.Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, rack.Key, rack.Payload>({
  name: "Racks",
  retrieve: async ({ client, params }) => await client.hardware.racks.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.hardware.racks.retrieve(key),
  listeners: [
    {
      channel: rack.STATUS_CHANNEL_NAME,
      onChange: Sync.parsedHandler(rack.statusZ, async ({ changed, onChange }) =>
        onChange(changed.details.rack, (prev) =>
          prev == null ? prev : { ...prev, status: changed },
        ),
      ),
    },
    {
      channel: rack.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(rack.keyZ, async ({ changed, onChange, client }) =>
        onChange(changed, (await client.hardware.racks.retrieve(changed)).payload),
      ),
    },
    {
      channel: rack.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(rack.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
  ],
});
