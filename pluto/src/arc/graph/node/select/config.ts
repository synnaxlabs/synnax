// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod/v4";

export const configZ = z.object({ type: z.literal("select") });
export interface Config extends z.infer<typeof configZ> {}

export const defaultConfig = (): Config => ({ type: "select" });
