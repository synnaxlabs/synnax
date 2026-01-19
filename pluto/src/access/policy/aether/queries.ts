// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology, type Synnax } from "@synnaxlabs/client";

import { type Flux } from "@/flux";
import { type flux } from "@/flux/aether";
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "policies";

export interface FluxStore extends flux.UnaryStore<
  access.policy.Key,
  access.policy.Policy
> {}

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

export const cachedRetrieveForSubject = (store: FluxSubStore, subject: ontology.ID) => {
  const roleRels = store.relationships.get((r) =>
    ontology.matchRelationship(r, {
      from: { type: "role" },
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: subject,
    }),
  );
  const policyRels = store.relationships.get((r) =>
    roleRels.some((rr) =>
      ontology.matchRelationship(r, {
        from: rr.from,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: { type: "policy" },
      }),
    ),
  );
  const policyKeys = policyRels.map((r) => r.to.key);
  return store.policies.get(policyKeys);
};

export interface RetrieveForSubjectParams {
  subject: ontology.ID;
  client: Synnax;
  store: FluxSubStore;
}

export const retrieveForSubject = async ({
  client,
  store,
  subject,
}: RetrieveForSubjectParams): Promise<access.policy.Policy[]> => {
  let policies = cachedRetrieveForSubject(store, subject);
  if (policies.length > 0) return policies;
  policies = await client.access.policies.retrieve({ for: subject });
  store.policies.set(policies);
  for (const p of policies) {
    const roles = await client.ontology.retrieveParents(
      [access.policy.ontologyID(p.key)],
      { types: ["role"] },
    );
    roles.forEach((r) => {
      const rel = {
        from: r.id,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: access.policy.ontologyID(p.key),
      };
      store.relationships.set(ontology.relationshipToString(rel), rel);
      const subjectRel = {
        from: r.id,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: subject,
      };
      store.relationships.set(ontology.relationshipToString(subjectRel), subjectRel);
    });
  }
  return policies;
};
