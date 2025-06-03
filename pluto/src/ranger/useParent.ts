// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Ontology, Synnax } from "@synnaxlabs/pluto";

export const useParent = (rangeKey: ranger.Key): ranger.Range | null => {
  const parent = Ontology.useParents(ranger.ontologyID(rangeKey)).find(
    ({ id: { type } }) => type === ranger.ONTOLOGY_TYPE,
  );
  const client = Synnax.use();
  if (parent == null || client == null) return null;
  return client.ranges.sugarOntologyResource(parent);
};
