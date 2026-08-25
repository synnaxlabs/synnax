// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type access,
  type ontology,
  type Synnax,
  UnexpectedError,
  user,
} from "@synnaxlabs/client";

import { Flux } from "@/flux";

const PERMISSION_PLURAL_RESOURCE_NAME = "permissions";

export type PermissionsQuery = {
  subject?: ontology.ID;
  objects: ontology.ID | ontology.ID[];
  action: access.Action;
};

const retrieveCurrent = async (client: Synnax): Promise<user.User> => {
  const user = client.auth?.user;
  if (user == null) {
    await client.connect();
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

const resolveSubject = (client: Synnax, subject?: ontology.ID): ontology.ID | null => {
  if (subject != null) return subject;
  const u = client?.auth?.user;
  if (u == null) return null;
  return user.ontologyID(u.key);
};

export interface IsGrantedParams {
  client: Synnax | null;
  query: PermissionsQuery;
}

export const isGranted = ({
  client,
  query: { subject, objects, action },
}: IsGrantedParams): boolean => {
  if (client == null) return false;
  const sub = resolveSubject(client, subject);
  if (sub == null) return false;
  return client.access.granted.getCached(sub, { objects, action }) ?? false;
};

export interface IsGrantedExtensionParams extends Omit<IsGrantedParams, "query"> {}

const { useResult: useResultGranted } = Flux.createRetrieve<PermissionsQuery, boolean>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { subject, objects, action } }) => {
    subject = await resolveSubjectAsync(client, subject);
    if (subject == null) return false;
    return await client.access.granted.retrieve(subject, { objects, action });
  },
  onChange: ({ client, query: { subject, objects, action } }, handler) => {
    const sub = resolveSubject(client, subject);
    if (sub == null) return () => {};
    return client.access.granted.onChange(sub, { objects, action }, handler);
  },
  getCached: ({ client, query }) => {
    const sub = resolveSubject(client, query.subject);
    if (sub == null) return undefined;
    return client.access.granted.getCached(sub, query);
  },
});

export const useGranted = (query: PermissionsQuery): boolean =>
  useResultGranted(query).data ?? false;

export const useRetrieveGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, action: "retrieve" });

export const useUpdateGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, action: "update" });

export const useDeleteGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, action: "delete" });

export const useCreateGranted = (id: ontology.ID | ontology.ID[]): boolean =>
  useGranted({ objects: id, action: "create" });

export interface GrantedParams extends Omit<IsGrantedParams, "query"> {
  id: ontology.ID | ontology.ID[];
}

export const viewGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "retrieve" } });

export const updateGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "update" } });

export const deleteGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "delete" } });

export const createGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "create" } });

export type LoadPermissionsQuery = {
  subject?: ontology.ID;
};

/**
 * useEnsurePermissions suspends until the subject's policies are cached, so a surface
 * mounted below it never reads an empty policy set as a denial. A failed read throws
 * to the surrounding boundary; call useInvalidatePermissions before resetting it, or
 * the settled failure throws again on the next render.
 */
export const {
  useEnsure: useEnsurePermissions,
  useInvalidate: useInvalidatePermissions,
} = Flux.createRetrieve<LoadPermissionsQuery, access.policy.Policy[]>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => {
    const subject = await resolveSubjectAsync(client, query.subject);
    if (subject == null) return [];
    return await client.access.policies.retrieveForSubject(subject);
  },
  onChange: ({ client, query }, handler) => {
    const subject = resolveSubject(client, query.subject);
    if (subject == null) return () => {};
    return client.access.policies.onChange({ for: subject }, handler);
  },
  getCached: ({ client, query }) => {
    const subject = resolveSubject(client, query.subject);
    if (subject == null) return undefined;
    return client.access.policies.getCached({ for: subject });
  },
});
