// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type resource, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";

export const client = createTestClient();

/** Resolves the root Arcs directory the Core keeps under the ontology root. */
export const retrieveRootDirectoryID = async (client: Synnax): Promise<resource.ID> => {
  const [root] = await client.ontology.children.retrieve({
    ids: ontology.ROOT_ID,
    types: ["arc_directory"],
  });
  if (root == null) throw new Error("root Arcs directory not found");
  return root.id;
};
