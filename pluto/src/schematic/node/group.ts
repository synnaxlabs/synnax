// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Icon } from "@synnaxlabs/lyra/icon";
import { type group } from "@synnaxlabs/client";

import { GROUP as fittings } from "@/schematic/node/fittings/group";
import { GROUP as general } from "@/schematic/node/general/group";
import { GROUP as meters } from "@/schematic/node/meters/group";
import { GROUP as process } from "@/schematic/node/process/group";
import { GROUP as pumps } from "@/schematic/node/pumps/group";
import { type Variant } from "@/schematic/node/registry";
import { GROUP as safety } from "@/schematic/node/safety/group";
import { GROUP as valves } from "@/schematic/node/valves/group";
import { GROUP as vessels } from "@/schematic/node/vessels/group";

export interface Group extends group.Group {
  Icon: Icon.FC;
  symbols: Variant[];
}

export const GROUPS: Group[] = [
  general,
  vessels,
  valves,
  pumps,
  meters,
  process,
  safety,
  fittings,
];
