// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";

export interface RetrieveParams {
  key: workspace.Key;
}

export const retrieve = Flux.createRetrieve<RetrieveParams, workspace.Workspace>({
  name: "Workspace",
  retrieve: ({ params, client }) => client.workspaces.retrieve(params.key),
});

export interface ListParams {
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, workspace.Key, workspace.Workspace>({
  name: "Workspace",
  retrieve: async ({ client, params }) => await client.workspaces.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.workspaces.retrieve(key),
  listeners: [
    {
      channel: workspace.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        workspace.workspaceZ,
        async ({ onChange, changed }) => onChange(changed.key, changed),
      ),
    },
    {
      channel: workspace.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(workspace.keyZ, async ({ onDelete, changed }) =>
        onDelete(changed),
      ),
    },
  ],
});
