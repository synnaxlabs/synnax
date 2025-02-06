// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Name = string;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;

export const groupZ = z.object({ key: keyZ, name: z.string() });

export type Payload = z.infer<typeof groupZ>;

export type ParamAnalysisResult =
  | {
      single: true;
      variant: "keys";
      normalized: Keys;
      actual: Key;
    }
  | {
      single: true;
      variant: "names";
      normalized: Names;
      actual: Name;
    }
  | {
      single: false;
      variant: "keys";
      normalized: Keys;
      actual: Keys;
    }
  | {
      single: false;
      variant: "names";
      normalized: Names;
      actual: Names;
    };

export const analyzeParams = (groups: Params): ParamAnalysisResult => {
  const normal = toArray(groups) as Keys | Names;
  if (normal.length === 0) throw new Error("No groups specified");
  const isKey = keyZ.safeParse(normal[0]).success;
  return {
    single: !Array.isArray(groups),
    variant: isKey ? "keys" : "names",
    normalized: normal,
    actual: groups,
  } as const as ParamAnalysisResult;
};

export const ONTOLOGY_TYPE: ontology.ResourceType = "group";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
