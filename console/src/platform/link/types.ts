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
// synnax://cluster/<cluster-key>/<resource-type>/<resource-key> for another resource.
// The cluster names the target rather than a Core record, so a link opens through
// whichever Core the person receiving it reaches the cluster through.

export const PREFIX = `synnax://cluster/`;

// Connect resolves the cluster a link names to a connected client, switching the active
// Core and awaiting a successful connection if necessary.
export interface Connect {
  (clusterKey: string): Promise<Synnax>;
}

export interface HandlerParams {
  client: Synnax;
  key: string;
}

export interface Handler {
  (params: HandlerParams): Promise<void>;
}

// UseHandler is a hook that returns a Handler for a single resource type. Feature
// packages register one per resource type in a Registry, keyed by resource type.
export interface UseHandler {
  (): Handler;
}

// Registry maps a resource type (the segment after the cluster key in a deep link) to
// the hook that produces its Handler.
export type Registry = Record<string, UseHandler>;
