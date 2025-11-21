// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type ontology, user } from "@synnaxlabs/client";

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

export interface HasPermissionParams
  extends Flux.RetrieveParams<PermissionsQuery, FluxSubStore> {}

const hasPermission = async ({
  client,
  query: { subject, objects, actions },
  store,
}: HasPermissionParams): Promise<boolean> => {
  const userKey = client.auth?.user?.key;
  if (subject == null && userKey != null) subject = user.ontologyID(userKey);
  if (subject == null) return false;
  const req = { subject, objects, actions };
  const policies = await Policy.retrieveForSubject({ client, query: req, store });
  return access.allowRequest(req, policies);
};

export const { useRetrieve: useGranted } = Flux.createRetrieve<
  PermissionsQuery,
  boolean,
  FluxSubStore
>({ name: PERMISSION_PLURAL_RESOURCE_NAME, retrieve: hasPermission });
