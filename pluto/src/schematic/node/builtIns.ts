// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { Fittings } from "@/schematic/node/fittings";
import { Flowmeters } from "@/schematic/node/flowmeters";
import { General } from "@/schematic/node/general";
import { Process } from "@/schematic/node/process";
import { Pumps } from "@/schematic/node/pumps";
import { Safety } from "@/schematic/node/safety";
import { Valves } from "@/schematic/node/valves";
import { Vessels } from "@/schematic/node/vessels";

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS: ReadonlyMap<string, schematic.symbol.Spec> = new Map(
  Object.entries({
    ...Fittings.BUILT_INS,
    ...Flowmeters.BUILT_INS,
    ...General.BUILT_INS,
    ...Process.BUILT_INS,
    ...Pumps.BUILT_INS,
    ...Safety.BUILT_INS,
    ...Valves.BUILT_INS,
    ...Vessels.BUILT_INS,
  }),
);
