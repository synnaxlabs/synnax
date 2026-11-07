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
  query,
  type resource,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";

import { Flux } from "@/flux";

const PERMISSION_PLURAL_RESOURCE_NAME = "permissions";

export type PermissionsQuery = {
  objects: resource.ID | resource.ID[];
  action: access.Action;
};

const subjectOf = (client: Synnax): resource.ID | null => {
  const u = client.users.current.getCached();
  return query.isLive(u) ? user.resourceID(u.key) : null;
};

const retrieveSubject = async (client: Synnax): Promise<resource.ID> =>
  user.resourceID((await client.users.current.retrieve()).key);

/**
 * Subscribes for the signed-in user, and moves the subscription each time a different
 * user signs in. Calls onSwitch after each move.
 */
const subscribeForCurrent = (
  client: Synnax,
  subscribe: (subject: resource.ID) => destructor.Destructor,
  onSwitch: () => void,
): destructor.Destructor => {
  let subject = subjectOf(client);
  let release = subject == null ? undefined : subscribe(subject);
  const stop = client.users.current.onChange(() => {
    const next = subjectOf(client);
    if (next?.key === subject?.key) return;
    release?.();
    subject = next;
    release = next == null ? undefined : subscribe(next);
    onSwitch();
  });
  return () => {
    stop();
    release?.();
  };
};

export interface IsGrantedParams {
  client: Synnax | null;
  query: PermissionsQuery;
}

export const isGranted = ({ client, query }: IsGrantedParams): boolean => {
  if (client == null) return false;
  const subject = subjectOf(client);
  if (subject == null) return false;
  return client.access.granted.getCached(subject, query) ?? false;
};

export interface IsGrantedExtensionParams extends Omit<IsGrantedParams, "query"> {}

const { useResult: useResultGranted } = Flux.createRetrieve<PermissionsQuery, boolean>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  // A read that never lands reads as a denial, so a moment of dead connection would
  // lock the subject out of surfaces it can in fact reach.
  retryUnreachable: true,
  retrieve: async ({ client, query }) =>
    await client.access.granted.retrieve(await retrieveSubject(client), query),
  onChange: ({ client, query }, handler) =>
    subscribeForCurrent(
      client,
      (subject) => client.access.granted.onChange(subject, query, handler),
      () => handler(undefined),
    ),
  getCached: ({ client, query }) => {
    const subject = subjectOf(client);
    if (subject == null) return undefined;
    return client.access.granted.getCached(subject, query);
  },
});

export const useGranted = (query: PermissionsQuery): boolean =>
  useResultGranted(query).data ?? false;

export const useRetrieveGranted = (id: resource.ID | resource.ID[]): boolean =>
  useGranted({ objects: id, action: "retrieve" });

export const useUpdateGranted = (id: resource.ID | resource.ID[]): boolean =>
  useGranted({ objects: id, action: "update" });

export const useDeleteGranted = (id: resource.ID | resource.ID[]): boolean =>
  useGranted({ objects: id, action: "delete" });

export const useCreateGranted = (id: resource.ID | resource.ID[]): boolean =>
  useGranted({ objects: id, action: "create" });

export interface GrantedParams extends Omit<IsGrantedParams, "query"> {
  id: resource.ID | resource.ID[];
}

export const viewGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "retrieve" } });

export const updateGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "update" } });

export const deleteGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "delete" } });

export const createGranted = ({ id, ...rest }: GrantedParams): boolean =>
  isGranted({ ...rest, query: { objects: id, action: "create" } });

export type LoadPermissionsQuery = Record<string, never>;

/**
 * useEnsurePermissions suspends until the signed-in user's policies are cached, so a
 * surface mounted below it never reads an empty policy set as a denial. A failed read
 * throws to the surrounding boundary; call useInvalidatePermissions before resetting
 * it, or the settled failure throws again on the next render.
 */
export const {
  useEnsure: useEnsurePermissions,
  useInvalidate: useInvalidatePermissions,
} = Flux.createRetrieve<LoadPermissionsQuery, access.policy.Policy[]>({
  name: PERMISSION_PLURAL_RESOURCE_NAME,
  // No workspace renders until this read lands, so a moment of dead connection must not
  // settle into a wall the subject has to clear by hand.
  retryUnreachable: true,
  retrieve: async ({ client }) =>
    await client.access.policies.retrieve({
      where: { subjects: [await retrieveSubject(client)] },
    }),
  onChange: ({ client }, handler) =>
    subscribeForCurrent(
      client,
      (subject) =>
        client.access.policies.onChange({ where: { subjects: [subject] } }, handler),
      () => handler(undefined),
    ),
  getCached: ({ client }) => {
    const subject = subjectOf(client);
    if (subject == null) return undefined;
    return client.access.policies.getCached({ where: { subjects: [subject] } });
  },
});
