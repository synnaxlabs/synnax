// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record } from "@synnaxlabs/x";
import z from "zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;

export const viewZ = z.object({
  key: keyZ,
  name: z.string(),
  type: z.string(),
  query: record.unknownZ,
});
export interface View extends z.infer<typeof viewZ> {}

export const newZ = viewZ.extend({ key: keyZ.optional() });
export interface New extends z.infer<typeof newZ> {}
