// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "policies";

export interface FluxStore
  extends flux.UnaryStore<access.policy.Key, access.policy.Policy> {}

export interface FluxSubStore extends Ontology.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
}

export const FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  access.policy.Key,
  access.policy.Policy
> = {
  listeners: [],
};
