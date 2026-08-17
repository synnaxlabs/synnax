// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type ontology, type Synnax as Client, user } from "@synnaxlabs/client";
import { createTestClientWithPolicy } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";

// Every console surface resolves the subject and its policies before it renders.
const BASELINE = [
  user.TYPE_ONTOLOGY_ID,
  access.role.TYPE_ONTOLOGY_ID,
  access.policy.TYPE_ONTOLOGY_ID,
];

/**
 * Creates a client whose only reads are the console baseline plus the given types, for
 * pinning a surface a retrieve grant withholds. A built-in role reads every type, so no
 * role can stand in.
 */
export const createTestClientWithReads = async (
  client: Client,
  ...types: ontology.ID[]
): Promise<Client> =>
  await createTestClientWithPolicy(client, {
    name: uuid.create(),
    objects: [...BASELINE, ...types],
    actions: ["retrieve"],
  });
