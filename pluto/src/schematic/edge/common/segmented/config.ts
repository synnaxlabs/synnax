// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import { z } from "zod";

import { type Segment, segmentZ } from "@/schematic/edge/common/segmented/connector";

export interface Config<V extends string = string> {
  variant: V;
  color: color.Color;
  segments: Segment[];
}

export const createConfigZ = <V extends string>(variant: V) =>
  z.object({
    variant: z.literal(variant),
    color: color.colorZ,
    segments: z.array(segmentZ),
  });

export const createDefaultConfig = <V extends string>(variant: V): Config<V> => ({
  variant,
  color: color.ZERO,
  segments: [],
});
