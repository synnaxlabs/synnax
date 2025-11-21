// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology, user } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import z from "zod";

import { type access as aetherAccess } from "@/access/aether";
import { Flux } from "@/flux";

export type Action = "create" | "delete" | "retrieve" | "update";

export interface PermissionsQuery {
  subject?: ontology.ID;
  objects: ontology.ID | ontology.ID[];
  actions: Action | Action[];
}

interface RetrievePoliciesForSubjectQuery {
  subject: ontology.ID;
}

const retrievePoliciesForSubject = async ({
  client,
  query: { subject },
  store,
}: Flux.RetrieveParams<
  RetrievePoliciesForSubjectQuery,
  aetherAccess.FluxSubStore
>): Promise<access.policy.Policy[]> => {
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

export interface HasPermissionParams
  extends Flux.RetrieveParams<PermissionsQuery, aetherAccess.FluxSubStore> {}

const hasPermission = async ({
  client,
  query: { subject, objects, actions },
  store,
}: HasPermissionParams): Promise<boolean> => {
  const userKey = client.auth?.user?.key;
  if (subject == null && userKey != null) subject = user.ontologyID(userKey);
  if (subject == null) return false;
  const req = { subject, objects, actions };
  const policies = await retrievePoliciesForSubject({ client, query: req, store });
  return access.allowRequest(req, policies);
};

export const { useRetrieve: useHasPermission } = Flux.createRetrieve<
  PermissionsQuery,
  boolean,
  aetherAccess.FluxSubStore
>({
  name: "Permissions",
  retrieve: hasPermission,
});

const roleFormSchema = z.object({
  key: z.uuid().optional(),
  name: z.string(),
  description: z.string().optional(),
});

export interface RetrieveRoleQuery {
  key: string;
}

const retrieveSingleRole = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<
  RetrieveRoleQuery,
  aetherAccess.FluxSubStore
>): Promise<access.role.Role> => {
  let r = store.roles.get(key);
  if (r != null) return r;
  r = await client.access.roles.retrieve({ key });
  store.roles.set(key, r);
  return r;
};

export const useRoleForm = Flux.createForm<
  Partial<RetrieveRoleQuery>,
  typeof roleFormSchema,
  aetherAccess.FluxSubStore
>({
  name: "Role",
  schema: roleFormSchema,
  initialValues: {
    key: undefined,
    name: "",
    description: "",
  },
  retrieve: async ({ client, query, store }) => {
    if (query.key == null) return;
    const role = await retrieveSingleRole({ client, query: { key: query.key }, store });
    store.roles.set(query.key, role);
  },
  update: async ({ client, value, store, set, rollbacks }) => {
    const v = value();
    let r: access.role.Role = { key: uuid.create(), ...v };
    rollbacks.push(store.roles.set(r.key, r));
    r = await client.access.roles.create(r);
    set("key", r.key);
  },
});
