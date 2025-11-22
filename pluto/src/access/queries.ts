// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type ontology, type Synnax, user } from "@synnaxlabs/client";

import { Policy } from "@/access/policy";
import { type policy } from "@/access/policy/aether";
import { type role } from "@/access/role/aether";
import { Flux } from "@/flux";

export type Action = "create" | "delete" | "retrieve" | "update";

const PERMISSION_PLURAL_RESOURCE_NAME = "Permissions";

export interface PermissionsQuery {
  subject?: ontology.ID;
  objects: ontology.ID | ontology.ID[];
  actions: Action | Action[];
}

export interface FluxSubStore extends role.FluxSubStore, policy.FluxSubStore {}

const resolveSubject = (
  client: Synnax,
  subject?: ontology.ID,
): ontology.ID | undefined => {
  if (subject != null) return subject;
  const userKey = client.auth?.user?.key;
  if (userKey != null) return user.ontologyID(userKey);
  return undefined;
};

export interface IsGrantedParams {
  store: FluxSubStore;
  client: Synnax | null;
  query: PermissionsQuery;
}

export interface IsGrantedExtensionParams extends Omit<IsGrantedParams, "query"> {}

export const isGranted = ({
  store,
  client,
  query: { subject, objects, actions },
}: IsGrantedParams) => {
  if (client == null) return false;
  subject = resolveSubject(client, subject);
  if (subject == null) return false;
  const policies = Policy.cachedRetrieveForSubject(store, subject);
  const req = { subject, objects, actions };
  return access.allowRequest(req, policies);
};

const { useRetrieve: useGrantedBase } = Flux.createRetrieve<
  PermissionsQuery,
  boolean,
  FluxSubStore
>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  retrieve: async ({
    client,
    query: { subject, objects, actions },
    store,
  }: Flux.RetrieveParams<PermissionsQuery, FluxSubStore>): Promise<boolean> => {
    subject = resolveSubject(client, subject);
    if (subject == null) return false;
    const policies = await Policy.retrieveForSubject({
      client,
      query: { subject },
      store,
    });
    return access.allowRequest({ subject, objects, actions }, policies);
  },
});

export const useGranted = (query: PermissionsQuery) => {
  const { data } = useGrantedBase(query);
  return data ?? false;
};

export interface LoadPermissionsQuery {
  subject?: ontology.ID;
}

export const { useRetrieve: useLoadPermissions } = Flux.createRetrieve<
  LoadPermissionsQuery,
  access.policy.Policy[],
  FluxSubStore
>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query, store }) => {
    const subject = resolveSubject(client, query.subject);
    if (subject == null) return [];
    return await Policy.retrieveForSubject({ client, query: { subject }, store });
  },
});
