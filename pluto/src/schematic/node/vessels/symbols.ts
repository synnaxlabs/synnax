// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CrossJunction } from "@/schematic/node/vessels/crossJunction";
import { Cylinder } from "@/schematic/node/vessels/cylinder";
import { Tank } from "@/schematic/node/vessels/tank";
import { TJunction } from "@/schematic/node/vessels/tJunction";

export const REGISTRY = {
  cross_junction: CrossJunction.spec,
  cylinder: Cylinder.spec,
  tank: Tank.spec,
  t_junction: TJunction.spec,
} as const;
