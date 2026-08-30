// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type z } from "zod";

export type Config<V extends schematic.EdgeConfigType = schematic.EdgeConfigType> =
  Extract<schematic.EdgeConfig, { variant: V }>;

export const createConfigZ = <V extends schematic.EdgeConfigType>(
  variant: V,
): z.ZodType<Config<V>> => schematic.EDGE_CONFIG_SCHEMAS[variant];

export const createDefaultConfig = <V extends schematic.EdgeConfigType>(
  variant: V,
): Config<V> =>
  schematic.EDGE_CONFIG_SCHEMAS[variant].parse({
    variant,
    color: color.ZERO,
    segments: [],
  });
