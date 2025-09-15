// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export type UseDeleteArgs = schematic.Params;

export const { useUpdate: useDelete } = Flux.createUpdate({
  name: "Schematic",
  update: async ({ client, value, store }) => {
    await client.workspaces.schematic.delete(value);
  },
});
