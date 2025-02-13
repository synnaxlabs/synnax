// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const keyZ = z.number();
export type Key = z.infer<typeof keyZ>;

export const rackZ = z.object({ key: keyZ, name: z.string() });
export interface Payload extends z.infer<typeof rackZ> {}

export const newZ = rackZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "rack";
export type OntologyType = typeof ONTOLOGY_TYPE;
