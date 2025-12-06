// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  type ontology,
  type Synnax,
  UnexpectedError,
  user,
} from "@synnaxlabs/client";

import { policy } from "@/access/policy/aether";
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

const retrieveCurrent = async (client: Synnax): Promise<user.User> => {
  const user = client.auth?.user;
  if (user == null) {
    const res = await client.connectivity.check();
    if (res.error != null) throw res.error;
    if (client.auth?.user == null)
      throw new UnexpectedError(
        "Expected user to be available after successfully connecting to cluster",
      );
    return client.auth.user;
  }
  return user;
};

const resolveSubjectAsync = async (
  client: Synnax,
  subject?: ontology.ID,
): Promise<ontology.ID | undefined> => {
  if (subject != null) return subject;
  const u = await retrieveCurrent(client);
  return user.ontologyID(u.key);
};

const resolveSubject = (client: Synnax, subject?: ontology.ID): ontology.ID => {
  if (subject != null) return subject;
  const u = client?.auth?.user;
  if (u == null) throw new UnexpectedError("User not found");
  return user.ontologyID(u.key);
};

export interface IsGrantedParams {
  store: FluxSubStore;
  client: Synnax | null;
  query: PermissionsQuery;
}

export const isGranted = ({
  store,
  client,
  query: { subject, objects, actions },
}: IsGrantedParams): boolean => {
  if (client == null) return false;
  const sub = resolveSubject(client, subject);
  const policies = policy.cachedRetrieveForSubject(store, sub);
  return access.allowRequest({ subject: sub, objects, actions }, policies);
};

export interface IsGrantedExtensionParams extends Omit<IsGrantedParams, "query"> {}

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
    subject = await resolveSubjectAsync(client, subject);
    if (subject == null) return false;
    const policies = await policy.retrieveForSubject({ client, subject, store });
    return access.allowRequest({ subject, objects, actions }, policies);
  },
});

export const useGranted = (query: PermissionsQuery) =>
  useGrantedBase(query)?.data ?? false;

export const VIEW_ACTIONS: Action[] = ["retrieve"];
export const DELETE_ACTIONS: Action[] = ["delete"];
export const CREATE_ACTIONS: Action[] = ["create"];
export const EDIT_ACTIONS: Action[] = ["update"];

export const useViewGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, actions: VIEW_ACTIONS });

export const useEditGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, actions: EDIT_ACTIONS });

export const useDeleteGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, actions: DELETE_ACTIONS });

export const useCreateGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, actions: CREATE_ACTIONS });

export interface GrantedParams extends Omit<IsGrantedParams, "query"> {
  id: ontology.ID | ontology.ID[];
}

export const viewGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, actions: VIEW_ACTIONS } });

export const editGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, actions: EDIT_ACTIONS } });

export const deleteGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, actions: DELETE_ACTIONS } });

export const createGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, actions: CREATE_ACTIONS } });

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
    const subject = await resolveSubjectAsync(client, query.subject);
    if (subject == null) return [];
    return await policy.retrieveForSubject({ client, subject, store });
  },
});
