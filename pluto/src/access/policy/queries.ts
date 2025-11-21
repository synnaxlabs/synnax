// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, ontology } from "@synnaxlabs/client";

import { type policy } from "@/access/policy/aether";
import { type Flux } from "@/flux";

interface RetrieveForSubjectQuery {
  subject: ontology.ID;
}

export const retrieveForSubject = async ({
  client,
  query: { subject },
  store,
}: Flux.RetrieveParams<RetrieveForSubjectQuery, policy.FluxSubStore>): Promise<
  access.policy.Policy[]
> => {
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
  if (policyRels.length > 0) {
    const policyKeys = policyRels.map((r) => r.to.key);
    return store.policies.get(policyKeys);
  }
  return await client.access.policies.retrieve({ for: subject });
};
