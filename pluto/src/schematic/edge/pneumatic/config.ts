// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";

import { Segmented } from "@/schematic/edge/common/segmented";

export const VARIANT = "pneumatic" as const;
export const NAME = "Pneumatic";
export const configZ = schematic.edgeConfigPneumaticZ;
export type Config = schematic.EdgeConfigPneumatic;
export const defaultConfig = (): Config => Segmented.createDefaultConfig(VARIANT);
