// Copyright 2024 Synnax Labs, Inc.
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

import { type access } from "@/access";
import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Name = string;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;

export const payloadZ = z.object({
  key: keyZ,
  name: z.string().min(1),
  timeRange: TimeRange.z,
  color: z.string().optional(),
});
export type Payload = z.infer<typeof payloadZ>;

export const newPayloadZ = payloadZ.extend({ key: z.string().uuid().optional() });
export type NewPayload = z.input<typeof newPayloadZ>;

export type ParamAnalysisResult =
  | {
      single: true;
      variant: "keys";
      normalized: Keys;
      actual: Key;
      empty: never;
    }
  | {
      single: true;
      variant: "names";
      normalized: Names;
      actual: Name;
      empty: never;
    }
  | {
      single: false;
      variant: "keys";
      normalized: Keys;
      actual: Keys;
      empty: boolean;
    }
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
  } as const as ParamAnalysisResult;
};

export const ONTOLOGY_TYPE: ontology.ResourceType = "range";
export const ALIAS_ONTOLOGY_TYPE: ontology.ResourceType = "range-alias";

export const rangeOntologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key });

export const rangeAliasOntologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ALIAS_ONTOLOGY_TYPE, key: key });

export const RESOLVE_ALIAS_ACTION: access.Action = "resolve";
