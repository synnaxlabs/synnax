// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { Segmented } from "@/schematic/edge/common/segmented";

export const VARIANT = "electric" as const;
export const NAME = "Electric signal";
export const configZ = Segmented.createConfigZ(VARIANT);
export type Config = z.infer<typeof configZ>;
export const defaultConfig = (): Config => Segmented.createDefaultConfig(VARIANT);
