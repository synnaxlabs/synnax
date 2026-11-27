// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import type { Names } from "@/schematic/node/builtIns";
import { CrossJunction } from "@/schematic/node/vessels/crossJunction";
import { Cylinder } from "@/schematic/node/vessels/cylinder";
import { Tank } from "@/schematic/node/vessels/tank";
import { TJunction } from "@/schematic/node/vessels/tJunction";

export const REGISTRY = {
  cylinder: Cylinder.spec,
  tank: Tank.spec,
} as const satisfies Partial<Record<schematic.NodeConfigType, unknown>>;

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  cross_junction: CrossJunction.SYMBOL,
  t_junction: TJunction.SYMBOL,
} as const satisfies Record<string, schematic.symbol.Spec>;

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  cross_junction: { name: "Cross junction", label: "" },
  t_junction: { name: "T junction", label: "" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
