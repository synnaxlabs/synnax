// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access } from "@synnaxlabs/client";

import { type Flux } from "@/flux";
import { type flux } from "@/flux/aether";
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "policies";

export interface FluxStore
  extends flux.UnaryStore<access.policy.Key, access.policy.Policy> {}

export interface FluxSubStore extends Ontology.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_LISTENER: Flux.ChannelListener<FluxSubStore, typeof access.policy.policyZ> = {
  channel: access.policy.SET_CHANNEL_NAME,
  schema: access.policy.policyZ,
  onChange: ({ store, changed }) => store.policies.set(changed.key, changed),
};

const DELETE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof access.policy.keyZ> = {
  channel: access.policy.DELETE_CHANNEL_NAME,
  schema: access.policy.keyZ,
  onChange: ({ store, changed }) => store.policies.delete(changed),
};

export const FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  access.policy.Key,
  access.policy.Policy
> = {
  listeners: [SET_LISTENER, DELETE_LISTENER],
};
