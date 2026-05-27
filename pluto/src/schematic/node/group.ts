// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group } from "@synnaxlabs/client";

import { type Icon } from "@/icon";
import { Fittings } from "@/schematic/node/fittings";
import { Flowmeters } from "@/schematic/node/flowmeters";
import { General } from "@/schematic/node/general";
import { Process } from "@/schematic/node/process";
import { Pumps } from "@/schematic/node/pumps";
import { type Variant } from "@/schematic/node/registry";
import { Safety } from "@/schematic/node/safety";
import { Valves } from "@/schematic/node/valves";
import { Vessels } from "@/schematic/node/vessels";

export interface Group extends group.Group {
  Icon: Icon.FC;
  symbols: Variant[];
}

export const GROUPS: Group[] = [
  General.GROUP,
  Vessels.GROUP,
  Valves.GROUP,
  Pumps.GROUP,
  Flowmeters.GROUP,
  Process.GROUP,
  Safety.GROUP,
  Fittings.GROUP,
];
