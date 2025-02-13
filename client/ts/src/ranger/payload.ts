// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export const nameZ = z.string().min(1);
export type Name = z.infer<typeof nameZ>;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;

export const payloadZ = z.object({
  key: keyZ,
  name: nameZ,
  timeRange: TimeRange.z,
  color: z.string().optional(),
});
export interface Payload extends z.infer<typeof payloadZ> {}

export const newZ = payloadZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export type ParamAnalysisResult =
  | { single: true; variant: "keys"; normalized: Keys; actual: Key; empty: never }
  | { single: true; variant: "names"; normalized: Names; actual: Name; empty: never }
  | { single: false; variant: "keys"; normalized: Keys; actual: Keys; empty: boolean }
  | {
      single: false;
      variant: "names";
      normalized: Names;
      actual: Names;
      empty: boolean;
    };

export const analyzeParams = (ranges: Params): ParamAnalysisResult => {
  const normal = toArray(ranges) as Keys | Names;
  const empty = normal.length === 0;
  let isKey = false;
  if (!empty) isKey = keyZ.safeParse(normal[0]).success;
  return {
    single: !Array.isArray(ranges),
    variant: isKey ? "keys" : "names",
    normalized: normal,
    actual: ranges,
    empty,
  } as ParamAnalysisResult;
};

export const ONTOLOGY_TYPE = "range";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ALIAS_ONTOLOGY_TYPE = "range-alias";
export type AliasOntologyType = typeof ALIAS_ONTOLOGY_TYPE;
