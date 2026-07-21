// Copyright 2026 Synnax Labs, Inc.
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

import { Flux } from "@/flux";

const PERMISSION_PLURAL_RESOURCE_NAME = "Permissions";

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
  const cached = client.access.policies.getCached({ for: sub });
  if (cached?.variant !== "changed") return false;
  return access.allowRequest({ subject: sub, objects, action }, cached.data);
};

export interface IsGrantedExtensionParams extends Omit<IsGrantedParams, "query"> {}

// Warms the relationship cache alongside the policies so role links are
// available without refetching.
const retrieveForSubject = async (
  client: Synnax,
  subject: ontology.ID,
): Promise<access.policy.Policy[]> => {
  const policies = await client.access.policies.retrieve({ for: subject });
  await client.ontology.retrieveParents(subject, { types: ["role"] });
  await Promise.all(
    policies.map(
      async (p) =>
        await client.ontology.retrieveParents(access.policy.ontologyID(p.key), {
          types: ["role"],
        }),
    ),
  );
  return policies;
};

const { useRetrieve: useGrantedBase } = Flux.createRetrieve<PermissionsQuery, boolean>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { subject, objects, action } }) => {
    subject = await resolveSubjectAsync(client, subject);
    if (subject == null) return false;
    const policies = await retrieveForSubject(client, subject);
    return access.allowRequest({ subject, objects, action }, policies);
  },
  subscribe: ({ client, query: { subject, objects, action } }, handler) => {
    const sub = resolveSubject(client, subject);
    if (sub == null) return () => {};
    const evaluate = (policies: access.policy.Policy[]): boolean =>
      access.allowRequest({ subject: sub, objects, action }, policies);
    const cached = client.access.policies.getCached({ for: sub });
    // Only notify when the grant itself flips: policy churn that cannot
    // change the answer must not re-render consumers.
    let prev = cached?.variant === "changed" ? evaluate(cached.data) : undefined;
    return client.access.policies.onChange({ for: sub }, (result) => {
      if (result?.variant !== "changed") return;
      const next = evaluate(result.data);
      if (next === prev) return;
      prev = next;
      handler({ variant: "changed", data: next });
    });
  },
});

export const useGranted = (query: PermissionsQuery) =>
  useGrantedBase(query)?.data ?? false;

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

export const { useRetrieve: useLoadPermissions } = Flux.createRetrieve<
  LoadPermissionsQuery,
  access.policy.Policy[]
>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => {
    const subject = await resolveSubjectAsync(client, query.subject);
    if (subject == null) return [];
    return await retrieveForSubject(client, subject);
  },
  subscribe: ({ client, query }, handler) => {
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
