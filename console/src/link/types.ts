// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { ontology, type Synnax } from "@synnaxlabs/client";

import { type Layout } from "@/layout";

// Links have the form synnax://cluster/<cluster-key> for a cluster or
// synnax://cluster/<cluster-key>/<resource-type>/<resource-key> for another resource

export const PREFIX = `synnax://${ontology.CLUSTER_TYPE}/`;

export interface HandlerProps {
  client: Synnax;
  dispatch: Dispatch<UnknownAction>;
  key: string;
  placeLayout: Layout.Placer;
}

export interface Handler {
  (props: HandlerProps): Promise<void>;
}
