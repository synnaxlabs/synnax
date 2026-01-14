// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Key } from "@/range/types.gen";

export const nameZ = z.string().min(1);
export type Name = z.infer<typeof nameZ>;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;
