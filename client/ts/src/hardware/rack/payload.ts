// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ontology } from "@/ontology";

export const rackKeyZ = z.number();

export type RackKey = z.infer<typeof rackKeyZ>;

export const rackZ = z.object({
  key: rackKeyZ,
  name: z.string(),
});

export type RackPayload = z.infer<typeof rackZ>;

export const newRackZ = rackZ.partial({ key: true });

export type NewRack = z.input<typeof newRackZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "rack";

export const ontologyID = (key: RackKey): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
