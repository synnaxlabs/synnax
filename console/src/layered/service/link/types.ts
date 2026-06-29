// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";

// Links have the form synnax://cluster/<cluster-key> for a cluster or
// synnax://cluster/<cluster-key>/<resource-type>/<resource-key> for another resource

export const PREFIX = `synnax://cluster/`;

// ClusterConnect resolves the cluster identified by key to a connected client,
// switching the active cluster and awaiting a successful connection if necessary.
export interface ClusterConnect {
  (key: string): Promise<Synnax>;
}

export interface HandlerArgs {
  client: Synnax;
  key: string;
}

export interface Handler {
  (args: HandlerArgs): Promise<void>;
}
