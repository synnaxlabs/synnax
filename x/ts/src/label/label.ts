// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;

export const labelZ = z.object({
  key: keyZ,
  name: z.string().min(1),
  color: z.string(),
});
export interface Label extends z.infer<typeof labelZ> {}
