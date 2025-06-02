// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, zod } from "@synnaxlabs/x";
import { TimeStamp } from "@synnaxlabs/x/telem";
import { z } from "zod";

export const keyZ = z.uint32();
export type Key = z.infer<typeof keyZ>;

export const stateZ = z.object({
  key: keyZ,
  variant: status.variantZ,
  message: z.string(),
  lastReceived: TimeStamp.z,
});

export interface State extends z.infer<typeof stateZ> {}

export const rackZ = z.object({
  key: keyZ,
  name: z.string(),
  state: zod.nullToUndefined(stateZ),
});

export interface Payload extends z.infer<typeof rackZ> {}

export const newZ = rackZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "rack";
export type OntologyType = typeof ONTOLOGY_TYPE;
