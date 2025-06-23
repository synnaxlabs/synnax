// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";

import { Query } from "@/query";

export const useSetSynchronizer = (onSet: (ws: workspace.Workspace) => void): void =>
  Query.useParsedListener(workspace.SET_CHANNEL_NAME, workspace.workspaceZ, onSet);

export const useDeleteSynchronizer = (onDelete: (ws: workspace.Key) => void): void =>
  Query.useParsedListener(workspace.DELETE_CHANNEL_NAME, workspace.keyZ, onDelete);
