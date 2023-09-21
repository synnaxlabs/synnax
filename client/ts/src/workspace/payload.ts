// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const keyZ = z.string().uuid();

export type Key = z.infer<typeof keyZ>;

export type Params = Key | Key[];

export const workspaceZ = z.object({
  name: z.string(),
  description: z.string(),
  key: keyZ,
});

export type Workspace = z.infer<typeof workspaceZ>;
