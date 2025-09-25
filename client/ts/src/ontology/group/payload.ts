// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type ID as OntologyID } from "@/ontology/payload";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export const nameZ = z.string();
export type Name = z.infer<typeof nameZ>;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;
export const groupZ = z.object({ key: keyZ, name: nameZ });
export interface Group extends z.infer<typeof groupZ> {}

export const ontologyID = (key: Key): OntologyID => ({ type: "group", key });
