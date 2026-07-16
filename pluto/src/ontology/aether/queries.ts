// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";

export interface RelationshipFluxStore extends flux.UnaryStore<
  string,
  ontology.Relationship
> {}

export interface ResourceFluxStore extends flux.UnaryStore<string, ontology.Resource> {}

export const RELATIONSHIPS_FLUX_STORE_KEY = "relationships";
export const RESOURCES_FLUX_STORE_KEY = "resources";

export interface FluxSubStore extends flux.Store {
  [RELATIONSHIPS_FLUX_STORE_KEY]: RelationshipFluxStore;
  [RESOURCES_FLUX_STORE_KEY]: ResourceFluxStore;
}
