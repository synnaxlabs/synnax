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

export const roleZ = z.object({
  key: keyZ,
  name: z.string(),
  description: z.string().optional(),
  policies: keyZ.array(),
  internal: z.boolean(),
});

export type Role = z.infer<typeof roleZ>;

export const newRoleZ = roleZ.partial({ key: true, policies: true, internal: true });

export type NewRole = z.infer<typeof newRoleZ>;
