// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";

export const client = createTestClient();

/** Resolves the Arcs group the Core keeps under the ontology root. */
export const retrieveArcsGroupID = async (client: Synnax): Promise<ontology.ID> => {
  const children = await client.ontology.children.retrieve({ ids: ontology.ROOT_ID });
  const arcs = children.find((r) => r.name === "Arcs");
  if (arcs == null) throw new Error("Arcs group not found");
  return arcs.id;
};
