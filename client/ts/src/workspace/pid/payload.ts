// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownRecord, unknownRecordZ } from "@synnaxlabs/x";
import { z } from "zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const pidZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ,
  snapshot: z.boolean(),
});

export const pidRemoteZ = z.object({
  key: z.string(),
  name: z.string(),
  snapshot: z.boolean(),
  data: z.string().transform((s) => JSON.parse(s) as UnknownRecord),
});

export type PID = z.infer<typeof pidZ>;
