// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export interface ListParams extends Flux.Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, rack.Key, rack.Rack>({
  name: "Racks",
  retrieve: async ({ client, params }) => await client.hardware.racks.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.hardware.racks.retrieve(key),
});
