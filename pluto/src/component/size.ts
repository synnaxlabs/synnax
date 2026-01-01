// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const SIZES = ["tiny", "small", "medium", "large", "huge"] as const;
export const size = z.enum(SIZES);
export type Size = z.infer<typeof size>;

export const isSize = (value: unknown): value is Size => size.safeParse(value).success;
