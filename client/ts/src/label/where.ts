// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Where } from "@/label/types.gen";
import { type Lines } from "@/label/where.gen";
import { type ontology } from "@/ontology";

/**
 * @param labeled - Reports whether the resource with the given ID carries the label.
 * @returns the typed lines of a label Where.
 */
export const createLines = (
  labeled: (id: ontology.ID, key: Key) => boolean,
): Lines => ({
  for: (l, id) => labeled(id, l.key),
});

/** @returns the resources that the for lines of where name. */
export const forIDs = (where: Where): ontology.ID[] => [
  ...(where.for != null ? [where.for] : []),
  ...[...(where.and ?? []), ...(where.or ?? [])].flatMap(forIDs),
  ...(where.not != null ? forIDs(where.not) : []),
];
