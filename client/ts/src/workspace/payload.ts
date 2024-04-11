// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary } from "@synnaxlabs/x/binary";
import { type UnknownRecord, unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

export const keyZ = z.string().uuid();

export type Key = z.infer<typeof keyZ>;

export type Params = Key | Key[];

export const workspaceZ = z.object({
  name: z.string(),
  key: keyZ,
  layout: unknownRecordZ.or(
    z.string().transform((s) => binary.JSON_ECD.decodeString(s) as UnknownRecord),
  ),
});

export const workspaceRemoteZ = workspaceZ.omit({ layout: true }).extend({
  layout: z.string().transform((s) => binary.JSON_ECD.decodeString(s) as UnknownRecord),
});

export type Workspace = z.infer<typeof workspaceZ>;
