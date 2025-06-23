// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";

import { Ontology } from "@/ontology";
import { Synnax } from "@/synnax";

export const useRetrieveChildRanges = (key: ranger.Key): ranger.Range[] => {
  const children = Ontology.useChildren(ranger.ontologyID(key)).filter(
    ({ id: { type } }) => type === ranger.ONTOLOGY_TYPE,
  );
  const client = Synnax.use();
  if (client == null) return [];
  return children.map((child) => client.ranges.sugarOntologyResource(child));
};

export const useRetrieveParentRange = (key: ranger.Key): ranger.Range | null => {
  const parent = Ontology.useParents(ranger.ontologyID(key)).find(
    ({ id: { type } }) => type === ranger.ONTOLOGY_TYPE,
  );
  const client = Synnax.use();
  if (parent == null || client == null) return null;
  return client.ranges.sugarOntologyResource(parent);
};
