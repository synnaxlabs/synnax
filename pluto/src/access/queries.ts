// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";

import { type Flux } from "@/flux";

export const POLICIES_FLUX_STORE_KEY = "policies";
const POLICY_RESOURCE_NAME = "Policy";
const POLICY_PLURAL_RESOURCE_NAME = "Policies";

export const ROLES_FLUX_STORE_KEY = "roles";
const ROLE_RESOURCE_NAME = "Role";
const ROLE_PLURAL_RESOURCE_NAME = "Roles";

export interface RoleFluxStore
  extends Flux.UnaryStore<access.role.Key, access.role.Role> {}

export interface PolicyFluxStore
  extends Flux.UnaryStore<access.policy.Key, access.policy.Policy> {}

export interface FluxSubStore extends Flux.Store {
  [POLICIES_FLUX_STORE_KEY]: PolicyFluxStore;
  [ROLES_FLUX_STORE_KEY]: RoleFluxStore;
}

export const ROLES_FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  access.role.Key,
  access.role.Role
> = {
  listeners: [],
};
