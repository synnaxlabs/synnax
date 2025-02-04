// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const ComponentSizes = ["small", "medium", "large", "huge"] as const;
export const componentSize = z.enum(ComponentSizes);
export type ComponentSize = z.infer<typeof componentSize>;

export const isComponentSize = (value: unknown): value is ComponentSize =>
  componentSize.safeParse(value).success;
