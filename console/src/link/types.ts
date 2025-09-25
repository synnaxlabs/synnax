// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type Store, type UnknownAction } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";

import { type Layout } from "@/layout";
import { type RootState } from "@/store";

// Links have the form synnax://cluster/<cluster-key> for a cluster or
// synnax://cluster/<cluster-key>/<resource-type>/<resource-key> for another resource

export const PREFIX = `synnax://cluster/`;

export interface ClusterHandlerArgs {
  store: Store<RootState>;
  key: string;
}

export interface ClusterHandler {
  (args: ClusterHandlerArgs): Promise<Synnax>;
}

export interface HandlerArgs {
  client: Synnax;
  dispatch: Dispatch<UnknownAction>;
  key: string;
  placeLayout: Layout.Placer;
}

export interface Handler {
  (args: HandlerArgs): Promise<void>;
}
