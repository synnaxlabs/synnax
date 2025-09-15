// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type linePlot } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export type UseDeleteArgs = linePlot.Params;

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs>({
  name: "LinePlot",
  update: async ({ client, value }) =>
    await client.workspaces.linePlot.delete(value),
});
