// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, schematic } from "@synnaxlabs/client";

import { Data } from "@/schematic/edge/data";
import { Electric } from "@/schematic/edge/electric";
import { Hydraulic } from "@/schematic/edge/hydraulic";
import { Jacketed } from "@/schematic/edge/jacketed";
import { Pipe } from "@/schematic/edge/pipe";
import { Pneumatic } from "@/schematic/edge/pneumatic";
import { Secondary } from "@/schematic/edge/secondary";
import { type Spec } from "@/schematic/edge/spec";

export const REGISTRY = {
  pipe: Pipe.spec,
  electric: Electric.spec,
  secondary: Secondary.spec,
  jacketed: Jacketed.spec,
  hydraulic: Hydraulic.spec,
  pneumatic: Pneumatic.spec,
  data: Data.spec,
} as const satisfies Record<schematic.EdgeConfigType, unknown>;

export const variantZ = schematic.edgeConfigTypeZ;
export type Variant = schematic.EdgeConfigType;

export const configZ = schematic.edgeConfigZ;
export type Config = schematic.EdgeConfig;

export const resolveSpec = (variant: string): Spec<Variant, Config> => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null) throw new NotFoundError(`Edge with variant ${variant} not found`);
  return spec as unknown as Spec<Variant, Config>;
};
