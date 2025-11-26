// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;

export const userZ = z.object({
  key: keyZ,
  username: z.string().min(1, "Username is required"),
  firstName: z.string().default(""),
  lastName: z.string().default(""),
});

export interface User extends z.infer<typeof userZ> {}

export const newZ = userZ
  .partial({ key: true, firstName: true, lastName: true })
  .extend({ password: z.string().min(1) });
export interface New extends z.infer<typeof newZ> {}
