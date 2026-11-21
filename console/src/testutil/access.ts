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
  framer,
  type resource,
  type Synnax as Client,
  type SynnaxParams,
  user,
} from "@synnaxlabs/client";
import {
  createRequestHold,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";

// Every console surface resolves the subject and its policies before it renders, and
// the flux store needs the frame stream to receive any live update at all.
const BASELINE = [
  user.TYPE_RESOURCE_ID,
  access.role.TYPE_RESOURCE_ID,
  access.policy.TYPE_RESOURCE_ID,
  framer.TYPE_RESOURCE_ID,
];

/** The types the subject may act on, keyed by action. */
export type Grants = Partial<Record<access.Action, resource.ID[]>>;

/**
 * Creates a client granted exactly these actions on these types, on top of the reads
 * every console surface needs to render at all. Every built-in role covers whole
 * action sets at once, so none of them can isolate a single gate.
 * @param params - Further parameters of the new client.
 */
export const createTestClientWithGrants = async (
  client: Client,
  grants: Grants = {},
  params?: Partial<SynnaxParams>,
): Promise<Client> => {
  const { retrieve = [], ...rest } = grants;
  return await createTestClientWithPolicy(
    client,
    [
      {
        name: uuid.create(),
        objects: [...BASELINE, ...retrieve],
        actions: ["retrieve"],
      },
      ...Object.entries(rest).map(([action, objects]) => ({
        name: uuid.create(),
        objects,
        actions: [action as access.Action],
      })),
    ],
    undefined,
    params,
  );
};

export interface HeldGrants {
  client: Client;
  /** Sends the held policy reads on to the Core. */
  release: () => void;
}

/**
 * Creates a client as {@link createTestClientWithGrants} does, whose policy reads wait
 * until release is called. The reads that create the client are not held.
 */
export const createTestClientWithHeldGrants = async (
  client: Client,
  grants: Grants = {},
): Promise<HeldGrants> => {
  const hold = createRequestHold((target) =>
    target.endsWith("/access/policy/retrieve"),
  );
  const held = await createTestClientWithGrants(client, grants, {
    middleware: [hold.middleware],
  });
  hold.engage();
  return { client: held, release: hold.release };
};
