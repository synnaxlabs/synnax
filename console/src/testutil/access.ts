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
  type ontology,
  type Synnax as Client,
  user,
} from "@synnaxlabs/client";
import { createTestClientWithPolicy } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";

// Every console surface resolves the subject and its policies before it renders, and
// the flux store needs the frame stream to receive any live update at all.
const BASELINE = [
  user.TYPE_ONTOLOGY_ID,
  access.role.TYPE_ONTOLOGY_ID,
  access.policy.TYPE_ONTOLOGY_ID,
  framer.TYPE_ONTOLOGY_ID,
];

/** The types the subject may act on, keyed by action. */
export type Grants = Partial<Record<access.Action, ontology.ID[]>>;

/**
 * Creates a client granted exactly these actions on these types, on top of the reads
 * every console surface needs to render at all. Every built-in role covers whole
 * action sets at once, so none of them can isolate a single gate.
 */
export const createTestClientWithGrants = async (
  client: Client,
  grants: Grants = {},
): Promise<Client> => {
  const { retrieve = [], ...rest } = grants;
  return await createTestClientWithPolicy(client, [
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
  ]);
};
