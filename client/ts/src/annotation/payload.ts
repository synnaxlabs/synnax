// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x/telem";
import { z } from "zod/v4";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const annotationZ = z.object({
  key: keyZ,
  timeRange: TimeRange.z,
  message: z.string(),
});
export interface Annotation extends z.infer<typeof annotationZ> {}

export const newZ = annotationZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "annotation";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

export const ontologyIDsFromAnnotations = (annotations: Annotation[]): ontology.ID[] =>
  annotations.map((a) => ontologyID(a.key));
